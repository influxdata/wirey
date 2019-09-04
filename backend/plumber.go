package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"wirey/pkg/wireguard"

	"github.com/vishvananda/netlink"
)

const (
	ifnamesiz  = 16
	maxretries = 5
	retryttl   = time.Second * 5
)

const (
	errMaxRetriesReached      = "maximum number of connection retries reached"
	errEndpointFormatNotValid = "endpoint must be in format <ip>:<port>, like 192.168.1.3:3459"
	errInvalidEndpoint        = "endpoint provided is not valid"
	errInterfaceNameLength    = "the interface name size cannot be more than " + string(ifnamesiz)
	errPrivateKeyWriting      = "error writing private key file: %s"
	errPrivateKeyOpening      = "error opening private key file: %s"
	errAddressAlreadyTaken    = "address already taken: %s"
	errAddLink                = "error adding the wireguard link: %s"
	errIntConversionPort      = "error during port conversion to int: %s"
)

// Peer ...
type Peer struct {
	PublicKey  []byte
	Endpoint   string
	IP         *net.IP
	AllowedIPs []string
}

// Interface ...
type Interface struct {
	Backend      Backend
	Name         string
	PeerCheckTTL time.Duration
	LocalPeer    Peer
	privateKey   []byte
	retries      int
}

// NewInterface ...
func NewInterface(
	b Backend,
	ifname string,
	endpoint string,
	ipaddr string,
	privateKeyPath string,
	peerCheckTTL time.Duration,
	allowedIPs []string,
) (*Interface, error) {
	hostPort := strings.Split(endpoint, ":")
	if len(hostPort) != 2 {
		return nil, fmt.Errorf(errEndpointFormatNotValid)
	}

	if net.ParseIP(hostPort[0]) == nil {
		return nil, fmt.Errorf(errInvalidEndpoint)
	}

	if err := validatePort(hostPort[1]); err != nil {
		return nil, err
	}

	// Check that the passed interface name is ok for the kernel
	// https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git/tree/include/uapi/linux/if.h?h=v4.14.36#n33
	if len(ifname) > ifnamesiz {
		return nil, fmt.Errorf(errInterfaceNameLength)
	}

	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		privKey, err := wireguard.Genkey()
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(privateKeyPath, privKey, 0600)
		if err != nil {
			return nil, fmt.Errorf(errPrivateKeyWriting, err.Error())
		}
	}

	privKey, err := ioutil.ReadFile(privateKeyPath)

	if err != nil {
		return nil, fmt.Errorf(errPrivateKeyOpening, err.Error())
	}

	pubKey, err := wireguard.ExtractPubKey(privKey)
	if err != nil {
		return nil, err
	}
	ipnet := net.ParseIP(ipaddr)
	return &Interface{
		Backend:      b,
		Name:         ifname,
		PeerCheckTTL: peerCheckTTL,
		privateKey:   privKey,
		LocalPeer: Peer{
			PublicKey:  pubKey,
			IP:         &ipnet,
			Endpoint:   endpoint,
			AllowedIPs: allowedIPs,
		},
	}, nil
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

func (i *Interface) retryConnection(reason string) error {
	log.Printf("Retry connect, reason: %s", reason)
	time.Sleep(retryttl)
	i.retries = i.retries + 1
	if i.retries > maxretries-1 {
		return fmt.Errorf("%s: Last error: %s", errMaxRetriesReached, reason)
	}
	err := i.Connect()

	if err == nil {
		i.retries = 0
	}

	return err
}

// Connect ...
func (i *Interface) Connect() error {
	taken, err := i.addressAlreadyTaken()

	if err != nil {
		return i.retryConnection(err.Error())
	}

	if taken {
		return fmt.Errorf(errAddressAlreadyTaken, *i.LocalPeer.IP)
	}

	// Join
	err = i.Backend.Join(i.Name, i.LocalPeer)

	if err != nil {
		return err
	}

	peersSHA := ""
	allowedIps := ""

	for {
		workingPeers, err := i.Backend.GetPeers(i.Name)
		if err != nil {
			return i.retryConnection(fmt.Sprintf("problem during extraction of peers from the backend: %s", err.Error()))
		}

		// We don't change anything if the peers remain the same
		newPeersSHA := extractPeersSHA(workingPeers)
		if newPeersSHA == peersSHA {
			time.Sleep(i.PeerCheckTTL)
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
			return i.retryConnection(fmt.Sprintf(errAddLink, err.Error()))
		}

		// Add the actual address to the link
		addr, err := netlink.ParseAddr(fmt.Sprintf("%s/24", i.LocalPeer.IP.String()))
		if err != nil {
			return i.retryConnection(fmt.Sprintf("error parsing the new ip address: %s", err.Error()))
		}

		// Configure wireguard
		s := strings.Split(i.LocalPeer.Endpoint, ":")
		port, err := strconv.Atoi(s[1])
		if err != nil {
			return fmt.Errorf(errIntConversionPort, err.Error())
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

			if len(p.AllowedIPs) > 0 {
				allowedIps = fmt.Sprintf("%s/32,%s", p.IP.String(), strings.Join(p.AllowedIPs[:], ","))
			} else {
				allowedIps = fmt.Sprintf("%s/32", p.IP.String())
			}

			conf.Peers = append(conf.Peers, wireguard.Peer{
				PublicKey:  string(p.PublicKey),
				AllowedIPs: allowedIps,
				Endpoint:   p.Endpoint,
			})
		}

		_, err = wireguard.SetConf(i.Name, conf)

		if err != nil {
			return i.retryConnection(err.Error())
		}

		netlink.AddrAdd(wirelink, addr)

		// Up the link
		err = netlink.LinkSetUp(wirelink)
		if err != nil {
			return err
		}

		log.Println("Link up")
	}
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
