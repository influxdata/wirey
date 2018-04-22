package store

import "net"

type Peer struct {
	PublicKey  string
	Endpoint   string
	AllowedIPS []net.IPAddr
}

type Interface struct {
	Backend Backend
	Name    string
}

func (i *Interface) Peers() []Peer {
	return i.Backend.GetPeers()
}
