package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

const (
	etcdWireyPrefix = "/wirey"
)

type EtcdBackend struct {
	Client *clientv3.Client
}

func NewEtcdBackend(endpoints []string) (*EtcdBackend, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &EtcdBackend{
		Client: cli,
	}, nil
}

func (e *EtcdBackend) Join(ifname string, p Peer) error {
	pj, err := json.Marshal(p)

	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(e.Client)
	_, err = kvc.Put(ctx, fmt.Sprintf("%s/%s/%s", etcdWireyPrefix, ifname, p.PublicKey), string(pj))
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdBackend) Leave(ifname string, p Peer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(e.Client)
	_, err := kvc.Delete(ctx, fmt.Sprintf("%s/%s/%s", etcdWireyPrefix, ifname, p.PublicKey))
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdBackend) GetPeers(ifname string) ([]Peer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(e.Client)
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
