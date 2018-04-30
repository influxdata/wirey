package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/influxdata/wirey/backend"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
		i, err := backend.NewInterface(b, ifname, fmt.Sprintf("%s:%s", endpoint, endpointPort), ipAddr, privateKeyPath)

		if err != nil {
			log.Fatal(err)
		}

		log.Fatal(i.Connect())
	},
}

func backendFactory() (backend.Backend, error) {
	// etcd backend
	etcdBackend := viper.GetStringSlice("etcd")
	if len(etcdBackend) > 0 {
		b, err := backend.NewEtcdBackend(etcdBackend)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	httpBackend := viper.GetString("http")
	if len(httpBackend) != 0 {
		b, err := backend.NewHTTPBackend(httpBackend)
		if err != nil {
			return nil, err
		}
		httpBackendBasicAuth := viper.GetString("httpbasicauth")
		if len(httpBackendBasicAuth) > 2 {
			splitted := strings.Split(httpBackendBasicAuth, ":")
			username, password := splitted[0], splitted[1]
			b.BasicAuth = &backend.BasicAuth{
				Username: username,
				Password: password,
			}
		}
		return b, nil
	}

	return nil, fmt.Errorf("No storage backend selected, available backends: [etcd, http]")
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
	pflags.String("http", "", "the http backend endpoint to use as backend, see also httpbasicauth if you need basic authentication")
	pflags.String("httpbasicauth", "", "basic auth for the http backend, in form username:password")
	pflags.String("ifname", "wg0", "the name to use for the interface (must be the same in all the peers)")
	pflags.String("ipaddr", "", "the ip for this node inside the tunnel, e.g: 10.0.0.3")
	pflags.String("privatekeypath", "/etc/wirey/privkey", "the local path where to load the private key from, if empty, a private key will be generated.")

	rootCmd.MarkFlagRequired("endpoint")
	rootCmd.MarkFlagRequired("ipaddr")

	viper.BindPFlag("endpoint", pflags.Lookup("endpoint"))
	viper.BindPFlag("endpoint-port", pflags.Lookup("endpoint-port"))
	viper.BindPFlag("etcd", pflags.Lookup("etcd"))
	viper.BindPFlag("http", pflags.Lookup("http"))
	viper.BindPFlag("httpbasicauth", pflags.Lookup("httpbasicauth"))
	viper.BindPFlag("ifname", pflags.Lookup("ifname"))
	viper.BindPFlag("ipaddr", pflags.Lookup("ipaddr"))
	viper.BindPFlag("privatekeypath", pflags.Lookup("privatekeypath"))

	viper.SetEnvPrefix("wirey")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
