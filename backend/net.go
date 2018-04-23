package backend

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/fntlnz/wirey/pkg/wireguard"
	"github.com/vishvananda/netlink"
)

const ipclass = "10.0.0.0"
const ipnetmask = "255.0.0.0"

type Peer struct {
	PublicKey []byte
	Endpoint  string
	IP        *net.IP
}

type Interface struct {
	Backend    Backend
	Name       string
	privateKey []byte
	LocalPeer  Peer
}

func NewInterface(b Backend, ifname string, endpoint string) (*Interface, error) {
	if len(strings.Split(endpoint, ":")) != 2 {
		return nil, fmt.Errorf("endpoint must be in format <ip>:<port>, like 192.168.1.3:3459")
	}

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
			Endpoint:  endpoint,
		},
	}, nil
}

func (i *Interface) Connect() error {
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
		presentIPs = append(presentIPs, *p.IP)
	}
	ipnet, err := ipam(presentIPs)
	if err != nil {
		return err
	}

	i.LocalPeer.IP = ipnet

	// Add myself to the distributed backend
	err = i.Backend.Join(i.Name, i.LocalPeer)
	if err != nil {
		return err
	}

	// Add the actual address to the link
	addr, err := netlink.ParseAddr(fmt.Sprintf("%s/32", ipnet.String()))
	if err != nil {
		return err
	}

	// Configure wireguard
	s := strings.Split(i.LocalPeer.Endpoint, ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		return err
	}
	conf := wireguard.Configuration{
		Interface: wireguard.Interface{
			ListenPort: port,
			PrivateKey: string(i.privateKey),
		},
		Peers: []wireguard.Peer{},
	}

	for _, p := range peers {
		peer := wireguard.Peer{
			PublicKey:  string(p.PublicKey),
			AllowedIPs: "10.0.0.0/8", //TODO(fntlnz) this should compute the list comma separated
			Endpoint:   p.Endpoint,
		}
		conf.Peers = append(conf.Peers, peer)
	}

	_, err = wireguard.SetConf(i.Name, conf)

	if err != nil {
		return err
	}

	netlink.AddrAdd(wirelink, addr)

	// Up the link
	return netlink.LinkSetUp(wirelink)
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
