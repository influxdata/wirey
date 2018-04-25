package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/wirey/pkg/wireguard"
	"github.com/vishvananda/netlink"
)

const ifnamesiz = 16

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

func NewInterface(b Backend, ifname string, endpoint string, ipaddr string, privateKeyPath string) (*Interface, error) {
	hostPort := strings.Split(endpoint, ":")
	if len(hostPort) != 2 {
		return nil, fmt.Errorf("endpoint must be in format <ip>:<port>, like 192.168.1.3:3459")
	}

	if net.ParseIP(hostPort[0]) == nil {
		return nil, fmt.Errorf("endpoint provided is not valid")
	}

	if err := validatePort(hostPort[1]); err != nil {
		return nil, err
	}

	// Check that the passed interface name is ok for the kernel
	// https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git/tree/include/uapi/linux/if.h?h=v4.14.36#n33
	if len(ifname) > ifnamesiz {
		return nil, fmt.Errorf("the interface name size cannot be more than %d", ifnamesiz)
	}

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		privKey, err := wireguard.Genkey()
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(privateKeyPath, privKey, 0600)
		if err != nil {
			return nil, fmt.Errorf("error writing private key file: %s", err.Error())
		}
	}

	privKey, err := ioutil.ReadFile(privateKeyPath)

	if err != nil {
		return nil, fmt.Errorf("error opening private key file: %s", err.Error())
	}

	pubKey, err := wireguard.ExtractPubKey(privKey)
	if err != nil {
		return nil, err
	}
	ipnet := net.ParseIP(ipaddr)
	return &Interface{
		Backend:    b,
		Name:       ifname,
		privateKey: privKey,
		LocalPeer: Peer{
			PublicKey: pubKey,
			IP:        &ipnet,
			Endpoint:  endpoint,
		},
	}, nil
}

func checkLinkAlreadyConnected(name string, peers []Peer, localPeer Peer) bool {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return false
	}
	if link == nil {
		return false
	}

	for _, peer := range peers {
		if bytes.Equal(peer.PublicKey, localPeer.PublicKey) {
			// oh gosh, I have the interface but the link is down
			if link.Attrs().OperState != netlink.OperUp {
				// TODO(fntlnz): check here that the link type is wireguard?
				return false
			}
			// Well I am already connected
			return true
		}
	}
	return false
}

func extractPeersSHA(workingPeers []Peer) string {
	sort.Slice(workingPeers, func(i, j int) bool {
		comparison := bytes.Compare(workingPeers[i].PublicKey, workingPeers[j].PublicKey)
		if comparison > 0 {
			return true
		}
		return false
	})
	keys := ""
	for _, p := range workingPeers {
		// hash the full peer to verify if it changed
		peerj, _ := json.Marshal(p)
		peerh := sha256.New()
		peerh.Write(peerj)
		keys = fmt.Sprintf("%s%x", keys, peerh.Sum(nil))
	}

	// hash of all the peers
	h := sha256.New()
	h.Write([]byte(keys))

	return fmt.Sprintf("%x", h.Sum(nil))
}

func (i *Interface) addressAlreadyTaken() (bool, error) {
	peers, err := i.Backend.GetPeers(i.Name)
	if err != nil {
		return false, err
	}
	for _, p := range peers {
		if p.IP.Equal(*i.LocalPeer.IP) && !bytes.Equal(i.LocalPeer.PublicKey, p.PublicKey) {
			return true, nil
		}
	}
	return false, nil
}

func (i *Interface) Connect() error {
	taken, err := i.addressAlreadyTaken()

	if err != nil {
		return err
	}

	if taken {
		return fmt.Errorf("address already taken: %s", *i.LocalPeer.IP)
	}

	// Join
	err = i.Backend.Join(i.Name, i.LocalPeer)

	if err != nil {
		return err
	}

	peersSHA := ""
	for {
		workingPeers, err := i.Backend.GetPeers(i.Name)
		if err != nil {
			return err
		}

		// We don't change anything if the peers remain the same
		newPeersSHA := extractPeersSHA(workingPeers)
		if newPeersSHA == peersSHA {
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("The peer list changed, reconfiguring...")
		peersSHA = newPeersSHA

		log.Println("Delete old link")
		// delete any old link
		link, _ := netlink.LinkByName(i.Name)
		if link != nil {
			netlink.LinkDel(link)
		}

		// create the actual link
		wirelink := &netlink.GenericLink{
			LinkAttrs: netlink.LinkAttrs{
				Name: i.Name,
			},
			LinkType: "wireguard",
		}
		err = netlink.LinkAdd(wirelink)
		if err != nil {
			return fmt.Errorf("error adding the wireguard link: %s", err.Error())
		}

		// Add the actual address to the link
		addr, err := netlink.ParseAddr(fmt.Sprintf("%s/24", i.LocalPeer.IP.String()))
		if err != nil {
			return fmt.Errorf("error parsing the new ip address: %s", err.Error())
		}

		// Configure wireguard
		// TODO(fntlnz) how do we assign the external ip address?
		s := strings.Split(i.LocalPeer.Endpoint, ":")
		port, err := strconv.Atoi(s[1])
		if err != nil {
			return fmt.Errorf("error during port conversion to int: %s", err.Error())
		}
		conf := wireguard.Configuration{
			Interface: wireguard.Interface{
				ListenPort: port,
				PrivateKey: string(i.privateKey),
			},
			Peers: []wireguard.Peer{},
		}

		for _, p := range workingPeers {
			if bytes.Equal(p.PublicKey, i.LocalPeer.PublicKey) {
				continue
			}
			conf.Peers = append(conf.Peers, wireguard.Peer{
				PublicKey:  string(p.PublicKey),
				AllowedIPs: fmt.Sprintf("%s/32", p.IP.String()),
				Endpoint:   p.Endpoint,
			})
		}

		_, err = wireguard.SetConf(i.Name, conf)

		if err != nil {
			return err
		}

		netlink.AddrAdd(wirelink, addr)

		// Up the link
		err = netlink.LinkSetUp(wirelink)
		if err != nil {
			return err
		}

		log.Println("Link up")
	}

	return nil
}

func validatePort(port string) error {
	if port != "" {
		v, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		if v < 0 || v > 65535 {
			return fmt.Errorf("port not valid %q", port)
		}
	}
	return nil
}
