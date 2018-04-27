package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type BasicAuth struct {
	Username string
	Password string
}
type HTTPBackend struct {
	client    *http.Client
	baseurl   string
	BasicAuth *BasicAuth
}

func NewHTTPBackend(baseurl string) (*HTTPBackend, error) {
	var transportWithTimeout = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &HTTPBackend{
		client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: transportWithTimeout,
		},
		baseurl: baseurl,
	}, nil
}

func publicKeySHA256(key []byte) string {
	h := sha256.New()
	h.Write(key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (b *HTTPBackend) Join(ifname string, p Peer) error {
	joinURL := fmt.Sprintf("%s/%s/%s", b.baseurl, ifname, publicKeySHA256(p.PublicKey))

	jsonPeer, err := json.Marshal(p)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(jsonPeer)
	req, err := http.NewRequest("POST", joinURL, buf)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	if b.BasicAuth != nil {
		req.SetBasicAuth(b.BasicAuth.Username, b.BasicAuth.Password)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("request error during join: %s", err.Error())
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("the join http request gave an unexpected status code: %d", res.StatusCode)
	}
	return nil
}

func (b *HTTPBackend) GetPeers(ifname string) ([]Peer, error) {
	getPeersURL := fmt.Sprintf("%s/%s", b.baseurl, ifname)

	req, err := http.NewRequest("GET", getPeersURL, nil)
	if err != nil {
		return nil, err
	}

	if b.BasicAuth != nil {
		req.SetBasicAuth(b.BasicAuth.Username, b.BasicAuth.Password)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error during get peers: %s", err.Error())
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the get peers http request gave an unexpected status code: %d", res.StatusCode)
	}

	peers := []Peer{}
	err = json.NewDecoder(res.Body).Decode(&peers)

	if err != nil {
		return nil, fmt.Errorf("error decoding peers during get peers: %s", err.Error())
	}

	return peers, nil
}
