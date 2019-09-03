package backend

import (
	"encoding/json"
	"fmt"

	"wirey/pkg/utils"

	log "github.com/Sirupsen/logrus"
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
func NewConsulBackend(endpoint string, token string) (*ConsulBackend, error) {

	config := api.DefaultConfig()
	config.Address = endpoint

	if len(token) > 0 {
		config.Token = token
	}

	// check if TLS is required
	/*
		if p.options[kvdb.TransportScheme] == "https" {
			tlsConfig := &api.TLSConfig{
				CAFile:             p.options[kvdb.CAFileKey],
				CertFile:           p.options[kvdb.CertFileKey],
				KeyFile:            p.options[kvdb.CertKeyFileKey],
				Address:            p.options[kvdb.CAAuthAddress],
				InsecureSkipVerify: strings.ToLower(p.options[kvdb.InsecureSkipVerify]) == "true",
			}

			consulTLSConfig, err := api.SetupTLSConfig(tlsConfig)
			if err != nil {
				log.Fatal(err)
			}

			config.Scheme = p.options[kvdb.TransportScheme]
			config.HttpClient = new(http.Client)
			config.HttpClient.Transport = &http.Transport{
				TLSClientConfig: consulTLSConfig,
			}
		}
	*/

	log.Infof("Connecting to Consul on %s\n", config.Address)

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

	log.Debugf("Inserting key on %s/%s/%s\n", consulWireyPrefix, ifname, utils.PublicKeySHA256(p.PublicKey))

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

		log.Debugf("Detected endpoint (peer) with address %s\n", peer.Endpoint)

		peers = append(peers, peer)

	}

	return peers, nil
}
