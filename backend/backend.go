package backend

type Backend interface {
	Join(ifname string, peer Peer) error
	GetPeers(ifname string) ([]Peer, error)
}
