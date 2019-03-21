package backend

import (
	"io/ioutil"
	"log"

	"github.com/hashicorp/go-discover"
)

// DiscoverNodes ...
func DiscoverNodes() ([]string, error) {
	// support discovery for all supported providers
	cloudDiscover := discover.Discover{}

	// use ioutil.Discard for no log output
	log := log.New(ioutil.Discard, "", 0)

	cloudProvider := "provider=aws region=eu-west-1 ..."
	res, err := cloudDiscover.Addrs(cloudProvider, log)

	if err != nil {
		return nil, err
	}

	return res, nil
}
