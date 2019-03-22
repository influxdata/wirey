package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/influxdata/wirey/backend"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version ...
var Version string

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
		peerDiscoveryTTL, err := time.ParseDuration(viper.GetString("peerdiscoveryttl"))
		if err != nil {
			log.Fatalf("The passed duration cannot be parsed: %s", err.Error())
		}
		i, err := backend.NewInterface(
			b,
			ifname,
			fmt.Sprintf("%s:%s", endpoint, endpointPort),
			ipAddr,
			privateKeyPath,
			peerDiscoveryTTL,
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
	if len(consulBackend) > 0 {
		b, err := backend.NewConsulBackend(fmt.Sprintf("%s:%d", consulBackend, consulPortBackend))
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	pflags := rootCmd.PersistentFlags()
	pflags.String("endpoint", "", "endpoint for this machine, e.g: 192.168.1.3")
	pflags.String("endpoint-port", "2345", "endpoint port for this machine")
	pflags.StringSlice("etcd", nil, "array of etcd servers to connect to")
	pflags.Int("etcd-port", 2379, "etcd port number")
	pflags.IP("consul", nil, "consul server to connect to, e.g: 127.0.0.1")
	pflags.Int("consul-port", 8500, "consul port number")
	pflags.String("http", "", "the http backend endpoint to use as backend, see also httpbasicauth if you need basic authentication")
	pflags.Int("http-port", 80, "http port number")
	pflags.String("httpbasicauth", "", "basic auth for the http backend, in form username:password")
	pflags.String("ifname", "wg0", "the name to use for the interface (must be the same in all the peers)")
	pflags.String("ipaddr", "", "the ip for this node inside the tunnel, e.g: 10.0.0.3")
	pflags.String("peerdiscoveryttl", "30s", "the time to wait to discover new peers using the configured backend")
	pflags.String("privatekeypath", "/etc/wirey/privkey", "the local path where to load the private key from, if empty, a private key will be generated.")
	pflags.String("discover", "", "discover configuration from the provider. e.g: provider=aws region=eu-west-1 ... Check go-discover for all the options.")

	rootCmd.MarkFlagRequired("endpoint")
	rootCmd.MarkFlagRequired("ipaddr")

	viper.BindPFlag("endpoint", pflags.Lookup("endpoint"))
	viper.BindPFlag("endpoint-port", pflags.Lookup("endpoint-port"))
	viper.BindPFlag("etcd", pflags.Lookup("etcd"))
	viper.BindPFlag("etcd-port", pflags.Lookup("etcd-port"))
	viper.BindPFlag("consul", pflags.Lookup("consul"))
	viper.BindPFlag("consul-port", pflags.Lookup("consul-port"))
	viper.BindPFlag("http", pflags.Lookup("http"))
	viper.BindPFlag("http-port", pflags.Lookup("http-port"))
	viper.BindPFlag("httpbasicauth", pflags.Lookup("httpbasicauth"))
	viper.BindPFlag("ifname", pflags.Lookup("ifname"))
	viper.BindPFlag("ipaddr", pflags.Lookup("ipaddr"))
	viper.BindPFlag("privatekeypath", pflags.Lookup("privatekeypath"))
	viper.BindPFlag("peerdiscoveryttl", pflags.Lookup("peerdiscoveryttl"))
	viper.BindPFlag("discover", pflags.Lookup("discover"))

	viper.SetEnvPrefix("wirey")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
