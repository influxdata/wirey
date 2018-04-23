package backend

import (
	"bytes"
	"fmt"
	"net"

	"github.com/fntlnz/wirey/pkg/wireguard"
	"github.com/vishvananda/netlink"
)

const ipclass = "10.0.0.0"
const ipnetmask = "255.0.0.0"

type Peer struct {
	PublicKey []byte
	Endpoint  *string
	IP        *net.IP
}

type Interface struct {
	Backend    Backend
	Name       string
	privateKey []byte
	LocalPeer  Peer
}

func NewInterface(b Backend, ifname string) (*Interface, error) {
	privKey, err := wireguard.Genkey()
	if err != nil {
		return nil, err
	}
	pubKey, err := wireguard.ExtractPubKey(privKey)
	if err != nil {
		return nil, err
	}
	return &Interface{
		Backend:    b,
		Name:       ifname,
		privateKey: privKey,
		LocalPeer: Peer{
			PublicKey: pubKey,
			IP:        nil,
			Endpoint:  nil,
		},
	}, nil
}

func (i *Interface) Connect(privateKey string) error {
	peers, err := i.Backend.GetPeers(i.Name)
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(i.Name)
	if err != nil {
		return err
	}
	for _, peer := range peers {
		if bytes.Equal(peer.PublicKey, i.LocalPeer.PublicKey) {
			// oh gosh, I have the interface but the link is down
			if link != nil && link.Attrs().OperState != netlink.OperUp {
				// TODO(fntlnz): check here that the link type is wireguard?
				break
			}
			// Well I am already connected
			return nil
		}
	}

	//TODO: WARNING(who populated this freaking local peer?)

	// Add myself to the distributed backend
	err = i.Backend.Join(i.Name, i.LocalPeer)
	if err != nil {
		return err
	}

	// If the link already exist at this point
	// it must be because it was down, so we are going to recreate it
	if link != nil {
		netlink.LinkDel(link)
	}

	// create the actual link
	wirelink := &netlink.GenericLink{
		LinkType: "wireguard",
	}
	err = netlink.LinkAdd(wirelink)
	if err != nil {
		return err
	}

	// TODO(fntlnz): find a better way to do this.
	// This could fail if someone else takes the same ip address
	// in the same moment
	presentIPs := []net.IP{}
	for _, p := range peers {
		presentIPs = append(presentIPs, p.IP)
	}
	ipnet, err := ipam(presentIPs)
	if err != nil {
		return err
	}

	// Add the actual address to the link
	addr, err := netlink.ParseAddr(fmt.Sprintf("%s/32", ipnet.String()))
	if err != nil {
		return err
	}
	netlink.AddrAdd(wirelink, addr)

	err = netlink.LinkSetUp(wirelink)

	if err != nil {
		return err
	}

	return nil
}

func ipam(presentIPs []net.IP) (*net.IP, error) {
	ip := net.ParseIP(ipclass)

	next, err := nextIP(ip)

check:
	for _, cur := range presentIPs {
		if next.Equal(cur) {
			next, err = nextIP(*next)
			if err != nil {
				return nil, err
			}
			goto check
		}
	}
	return next, nil
}

func nextIP(current net.IP) (*net.IP, error) {
	current = current.To4()
	if current == nil {
		return nil, fmt.Errorf("the current ip address is not 4 octects")
	}

	ip := current.Mask(current.DefaultMask())
	ip[3]++
	return &ip, nil
}
