package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/influxdata/wirey/backend"
	"github.com/spf13/cobra"
)

var endpoint string
var endpointPort string
var ipAddr string
var privateKeyPath string
var ifname string
var etcdBackend []string
var httpBackend string
var httpBackendBasicAuth string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wirey",
	Short: "manage local wireguard interfaces in a distributed system",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		b, err := backendFactory()

		if err != nil {
			log.Fatal(err)
		}

		i, err := backend.NewInterface(b, ifname, fmt.Sprintf("%s:%s", endpoint, endpointPort), ipAddr, privateKeyPath)

		if err != nil {
			log.Fatal(err)
		}

		log.Fatal(i.Connect())
	},
}

func backendFactory() (backend.Backend, error) {
	// etcd backend
	if etcdBackend != nil {
		b, err := backend.NewEtcdBackend(etcdBackend)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// http backend with optional basic auth
	if len(httpBackend) != 0 {
		b, err := backend.NewHTTPBackend(httpBackend)
		if err != nil {
			return nil, err
		}
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
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "endpoint for this machine, e.g: 192.168.1.3")
	rootCmd.PersistentFlags().StringVar(&endpointPort, "endpoint-port", "2345", "endpoint port for this machine")
	rootCmd.PersistentFlags().StringVar(&ipAddr, "ipaddr", "", "the ip for this node inside the tunnel, e.g: 10.0.0.3")
	rootCmd.PersistentFlags().StringVar(&privateKeyPath, "privatekeypath", "/etc/wirey/privkey", "the local path where to load the private key from, if empty, a private key will be generated.")
	rootCmd.PersistentFlags().StringVar(&ifname, "ifname", "wg0", "the name to use for the interface (must be the same in all the peers)")
	rootCmd.PersistentFlags().StringVar(&httpBackend, "http", "", "the http backend endpoint to use as backend, see also httpbasicauth if you need basic authentication")
	rootCmd.PersistentFlags().StringVar(&httpBackendBasicAuth, "httpbasicauth", "", "basic auth for the http backend, in form username:password")
	rootCmd.PersistentFlags().StringArrayVar(&etcdBackend, "etcd", nil, "array of etcd servers to connect to")
}
