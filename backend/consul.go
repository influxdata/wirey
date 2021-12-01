package backend

import (
	"encoding/json"
	"fmt"

	"wirey/pkg/utils"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

const (
	consulWireyPrefix = "wirey"
)

// ConsulBackend ...
type ConsulBackend struct {
	client *api.Client
}

// NewConsulBackend ...
func NewConsulBackend(endpoint string, token string) (*ConsulBackend, error) {

	config := api.DefaultConfig()
	config.Address = endpoint

	if len(token) > 0 {
		config.Token = token
	}

	log.Infof("consul: connecting to %s\n", config.Address)

	cli, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	// check health to ensure communication with consul are working
	if _, _, err := cli.Health().State(api.HealthAny, nil); err != nil {
		log.Errorf("consul: health check failed for %v : %v", config.Address, err)
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

	log.Debugf("consul: inserting key on %s/%s/%s\n", consulWireyPrefix, ifname, utils.PublicKeySHA256(p.PublicKey))

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

		log.Debugf("consul: detected endpoint (peer) with address %s\n", peer.Endpoint)

		peers = append(peers, peer)

	}

	return peers, nil
}
