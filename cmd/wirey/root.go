package main

import (
	"fmt"
	"log"
	"os"

	"github.com/influxdata/wirey/backend"
	"github.com/spf13/cobra"
)

var endpoint string
var endpointPort string
var ipAddr string
var privateKeyPath string
var ifname string
var etcdBackend []string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wirey",
	Short: "manage local wireguard interfaces in a distributed system",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if etcdBackend == nil {
			log.Fatal("No storage backend selected, available backends: [etcd]")
		}

		b, err := backend.NewEtcdBackend(etcdBackend)
		if err != nil {
			log.Fatal(err)
		}

		i, err := backend.NewInterface(b, ifname, fmt.Sprintf("%s:%s", endpoint, endpointPort), ipAddr, privateKeyPath)

		if err != nil {
			log.Fatal(err)
		}

		log.Fatal(i.Connect())
		// this is not the intended way to user interface bt I need to test
	},
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
	rootCmd.PersistentFlags().StringArrayVar(&etcdBackend, "etcd", nil, "array of etcd servers to connect to")
}
