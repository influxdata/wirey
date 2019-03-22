package backend

import (
	"encoding/json"
	"fmt"

	"github.com/influxdata/wirey/pkg/utils"

	"github.com/hashicorp/consul/api"
)

const (
	consulWireyPrefix = "wirey"
)

// ConsulBackend ...
type ConsulBackend struct {
	client *api.Client
}

// NewConsulBackend ...
func NewConsulBackend(endpoint string) (*ConsulBackend, error) {

	config := api.DefaultConfig()
	config.Address = endpoint

	fmt.Printf("Connecting to Consul on %s\n", config.Address)

	cli, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ConsulBackend{
		client: cli,
	}, nil
}

// Join ...
func (e *ConsulBackend) Join(ifname string, p Peer) error {
	pj, err := json.Marshal(p)

	if err != nil {
		return err
	}

	kvc := e.client.KV()

	_, err = kvc.Put(
		&api.KVPair{
			Key:   fmt.Sprintf("%s/%s/%s", consulWireyPrefix, ifname, utils.PublicKeySHA256(p.PublicKey)),
			Value: pj,
		},
		nil,
	)

	if err != nil {
		return err
	}
	return nil
}

// GetPeers ...
func (e *ConsulBackend) GetPeers(ifname string) ([]Peer, error) {
	kvc := e.client.KV()
	res, _, err := kvc.List(fmt.Sprintf("%s/%s", consulWireyPrefix, ifname), nil)
	if err != nil {
		return nil, err
	}

	peers := []Peer{}

	if res == nil {
		return peers, nil
	}

	for _, v := range res {
		peer := Peer{}

		err = json.Unmarshal(v.Value, &peer)
		if err != nil {
			return nil, err
		}

		peers = append(peers, peer)

	}

	return peers, nil
}
