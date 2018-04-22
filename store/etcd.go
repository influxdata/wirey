package store

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type EtcdBackend struct {
	endpoints []string
}

func NewEtcdBackend(endpoints []string) (*EtcdBackend, error) {
	return &EtcdBackend{
		endpoints: endpoints,
	}, nil
}

func (e *EtcdBackend) Join(p Peer) error {
	fmt.Printf("Join pre")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   e.endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	kvc := clientv3.NewKV(cli)
	res, err := kvc.Put(ctx, "ciao", "sample_value")
	fmt.Printf("here")
	cancel()
	if err != nil {
		return err
	}

	fmt.Printf("%#-v", res)
	return nil
}

func (e *EtcdBackend) Leave(p Peer) error {
	return nil
}

func (e *EtcdBackend) GetPeers() []Peer {
	return []Peer{}
}
