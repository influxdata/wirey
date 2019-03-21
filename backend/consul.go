package backend

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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
func NewConsulBackend(endpoints []string) (*ConsulBackend, error) {

	config := api.DefaultConfig()
	config.Address = strings.Join(endpoints, " ")

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
			Key:   fmt.Sprintf("%s/%s/%s", consulWireyPrefix, ifname, url.QueryEscape(string(p.PublicKey))),
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
	_, _, err := kvc.Get(fmt.Sprintf("%s/%s", consulWireyPrefix, ifname), nil)
	if err != nil {
		return nil, err
	}

	peers := []Peer{}
	/*
		for _, v := range res.Key {
			fmt.Printf("%+v", v)

				peer := Peer{}

					err = json.Unmarshal(v.Value, &peer)
					if err != nil {
						return nil, err
					}
					peers = append(peers, peer)

		}
	*/
	return peers, nil
}
