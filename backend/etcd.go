package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
)

const (
	etcdWireyPrefix = "/wirey"
)

// EtcdBackend ...
type EtcdBackend struct {
	client *clientv3.Client
}

// NewEtcdBackend ...
func NewEtcdBackend(endpoints []string) (*EtcdBackend, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &EtcdBackend{
		client: cli,
	}, nil
}

// Join ...
func (e *EtcdBackend) Join(ifname string, p Peer) error {
	pj, err := json.Marshal(p)

	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(e.client)
	_, err = kvc.Put(ctx, fmt.Sprintf("%s/%s/%s", etcdWireyPrefix, ifname, p.PublicKey), string(pj))
	cancel()
	if err != nil {
		return err
	}
	return nil
}

// GetPeers ...
func (e *EtcdBackend) GetPeers(ifname string) ([]Peer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(e.client)
	res, err := kvc.Get(ctx, fmt.Sprintf("%s/%s", etcdWireyPrefix, ifname), clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	peers := []Peer{}
	for _, v := range res.Kvs {
		peer := Peer{}
		err = json.Unmarshal(v.Value, &peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}
	return peers, nil
}
