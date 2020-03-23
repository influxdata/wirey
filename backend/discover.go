package backend

import (
	"log"
	"os"

	"github.com/hashicorp/go-discover"
)

// DiscoverNodes ...
func DiscoverNodes(discoverConf string) ([]string, error) {
	// support discovery for all supported providers
	cloudDiscover := discover.Discover{}

	// use ioutil.Discard for no log output
	//log := log.New(ioutil.Discard, "", 0)
	log := log.New(os.Stderr, "", log.LstdFlags)

	res, err := cloudDiscover.Addrs(discoverConf, log)

	if err != nil {
		return nil, err
	}

	return res, nil
}
