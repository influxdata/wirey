package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wirey/backend"

	socktmpl "github.com/hashicorp/go-sockaddr/template"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version ...
var Version string

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "wirey",
	Short: "manage local wireguard interfaces in a distributed system",
	Run: func(cmd *cobra.Command, args []string) {
		b, err := backendFactory()

		if err != nil {
			log.Fatal(err)
		}

		privateKeyPath := viper.GetString("privatekeypath")
		privKeyBaseDir := filepath.Dir(privateKeyPath)
		if _, err := os.Stat(privKeyBaseDir); os.IsNotExist(err) {
			if err := os.Mkdir(privKeyBaseDir, 0600); err != nil {
				log.Fatalf("Unable to create the base directory for the wirey private key: %s - %s", privKeyBaseDir, err.Error())
			}
		}

		ifname := viper.GetString("ifname")
		endpoint := viper.GetString("endpoint")
		endpointPort := viper.GetString("endpoint-port")
		ipAddr := viper.GetString("ipaddr")

		// Endpoint
		endpoint, err = socktmpl.Parse(endpoint)
		if err != nil {
			log.Fatal(err)
		}

		// IP Address
		ipAddr, err = socktmpl.Parse(ipAddr)
		if err != nil {
			log.Fatal(err)
		}

		// Check peer discovery ttl
		peerDiscoveryTTL, err := time.ParseDuration(viper.GetString("peerdiscoveryttl"))
		if err != nil {
			log.Fatalf("The passed duration (peerdiscoveryttl) cannot be parsed: %s", err.Error())
		}

		// Allowed IPs
		allowedIps := viper.GetStringSlice("allowedips")
		allowedIpsList := make([]string, 0)

		for _, v := range allowedIps {
			_, _, err := net.ParseCIDR(v)

			if err != nil {
				log.Errorf("Not valid allowed ip. %s\n", err)
				continue
			}

			allowedIpsList = append(allowedIpsList, v)
		}

		i, err := backend.NewInterface(
			b,
			ifname,
			fmt.Sprintf("%s:%s", endpoint, endpointPort),
			ipAddr,
			privateKeyPath,
			peerDiscoveryTTL,
			allowedIps,
		)

		if err != nil {
			log.Fatal(err)
		}

		log.Fatal(i.Connect())
	},
}

func backendFactory() (backend.Backend, error) {

	etcdBackend := viper.GetStringSlice("etcd")
	etcdPortBackend := viper.GetInt("etcd-port")
	consulBackend := viper.GetString("consul")
	consulPortBackend := viper.GetInt("consul-port")
	consulAddressBackend := viper.GetString("consul-address")
	consulTokenBackend := viper.GetString("consul-token")
	httpBackend := viper.GetString("http")
	//httpPortBackend := viper.GetInt("http-port")
	discoverConf := viper.GetString("discover")

	// discover
	discoverHosts := []string{}
	if len(discoverConf) > 0 {
		discoverHosts, _ = backend.DiscoverNodes(discoverConf)

		// Replace ETCD Hosts with Discovered Hosts
		if len(etcdBackend) > 0 {
			for _, v := range discoverHosts {
				etcdBackend = append(etcdBackend, fmt.Sprintf("%s:%d", v, etcdPortBackend))
			}
		}

		// Replace Consul Hosts with Discovered Hosts
		if len(consulBackend) > 0 {
			rand.Seed(time.Now().UnixNano())

			// TODO: Consul discovered hosts should be health checked instead of choosing a random one
			consulBackend = discoverHosts[rand.Intn(len(discoverHosts)-1)]
		}

		// Replace HTTP Hosts with Discovered Hosts
		//if len(httpBackend) > 0 {
		//	for _, v := range discoverHosts {
		//		httpBackend = append(httpBackend, fmt.Sprintf("%s:%d", v, httpPortBackend))
		//	}
		//}
	}

	// etcd backend
	if len(etcdBackend) > 0 {
		b, err := backend.NewEtcdBackend(etcdBackend)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// consul backend
	if len(consulBackend) > 0 || len(consulAddressBackend) > 0 {

		consulBackend, err := socktmpl.Parse(consulBackend)
		if err != nil {
			log.Fatal(err)
		}

		if consulAddressBackend == "" {
			consulAddressBackend = fmt.Sprintf("%s:%d", consulBackend, consulPortBackend)
		}

		b, err := backend.NewConsulBackend(
			consulAddressBackend,
			consulTokenBackend,
		)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// http backend
	if len(httpBackend) != 0 {
		b, err := backend.NewHTTPBackend(httpBackend, Version)
		if err != nil {
			return nil, err
		}
		httpBackendBasicAuth := viper.GetString("httpbasicauth")
		if len(httpBackendBasicAuth) > 0 {
			splitted := strings.Split(httpBackendBasicAuth, ":")
			if len(splitted) != 2 {
				return nil, fmt.Errorf("the provided basic auth credentials are not in format username:password")
			}
			username, password := splitted[0], splitted[1]
			b.BasicAuth = &backend.BasicAuth{
				Username: username,
				Password: password,
			}
		}
		return b, nil
	}

	return nil, fmt.Errorf("No storage backend selected, available backends: [etcd, consul, http]")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// Initialize configuration file
	cobra.OnInitialize(initConfig)

	pflags := rootCmd.PersistentFlags()
	pflags.StringVar(&cfgFile, "config", "", "config file (default is ./wirey.yml)")
	pflags.String("endpoint", "", "endpoint for this machine, e.g: 192.168.1.3")
	pflags.String("endpoint-port", "2345", "endpoint port for this machine")
	pflags.StringSlice("etcd", nil, "array of etcd servers to connect to")
	pflags.Int("etcd-port", 2379, "etcd port number")
	pflags.String("consul", "", "consul server to connect to, e.g: 127.0.0.1")
	pflags.Int("consul-port", 8500, "consul port number")
	pflags.String("consul-address", "", "consul address (overrides host and port)")
	pflags.String("consul-token", "", "consul acl token")
	pflags.String("http", "", "the http backend endpoint to use as backend, see also httpbasicauth if you need basic authentication")
	pflags.Int("http-port", 80, "http port number")
	pflags.String("httpbasicauth", "", "basic auth for the http backend, in form username:password")
	pflags.String("ifname", "wg0", "the name to use for the interface (must be the same in all the peers)")
	pflags.String("ipaddr", "", "the ip for this node inside the tunnel, e.g: 10.0.0.3")
	pflags.String("peerdiscoveryttl", "30s", "the time to wait to discover new peers using the configured backend")
	pflags.String("privatekeypath", "/etc/wirey/privkey", "the local path where to load the private key from, if empty, a private key will be generated.")
	pflags.String("discover", "", "discover configuration from the provider. e.g: provider=aws region=eu-west-1 ... Check go-discover for all the options.")
	pflags.StringSlice("allowedips", nil, "array of allowed ips")

	rootCmd.MarkFlagRequired("endpoint")
	rootCmd.MarkFlagRequired("ipaddr")

	viper.BindPFlag("endpoint", pflags.Lookup("endpoint"))
	viper.BindPFlag("endpoint-port", pflags.Lookup("endpoint-port"))
	viper.BindPFlag("etcd", pflags.Lookup("etcd"))
	viper.BindPFlag("etcd-port", pflags.Lookup("etcd-port"))
	viper.BindPFlag("consul", pflags.Lookup("consul"))
	viper.BindPFlag("consul-port", pflags.Lookup("consul-port"))
	viper.BindPFlag("consul-address", pflags.Lookup("consul-address"))
	viper.BindPFlag("consul-token", pflags.Lookup("consul-token"))
	viper.BindPFlag("http", pflags.Lookup("http"))
	viper.BindPFlag("http-port", pflags.Lookup("http-port"))
	viper.BindPFlag("httpbasicauth", pflags.Lookup("httpbasicauth"))
	viper.BindPFlag("ifname", pflags.Lookup("ifname"))
	viper.BindPFlag("ipaddr", pflags.Lookup("ipaddr"))
	viper.BindPFlag("privatekeypath", pflags.Lookup("privatekeypath"))
	viper.BindPFlag("peerdiscoveryttl", pflags.Lookup("peerdiscoveryttl"))
	viper.BindPFlag("discover", pflags.Lookup("discover"))
	viper.BindPFlag("allowedips", pflags.Lookup("allowedips"))

	viper.SetEnvPrefix("wirey")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName("wirey")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Warn(err)
	}
}
