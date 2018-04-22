package store

type Backend interface {
	Join(Peer) error
	Leave(Peer) error
	GetPeers() []Peer
}
