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

const httpUserAgent = "wirey"

// BasicAuth ...
type BasicAuth struct {
	Username string
	Password string
}

// HTTPBackend ...
type HTTPBackend struct {
	client       *http.Client
	baseurl      string
	BasicAuth    *BasicAuth
	wireyVersion string
}

// NewHTTPBackend ...
func NewHTTPBackend(baseurl, wireyVersion string) (*HTTPBackend, error) {
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
		baseurl:      baseurl,
		wireyVersion: wireyVersion,
	}, nil
}

func publicKeySHA256(key []byte) string {
	h := sha256.New()
	h.Write(key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Join ...
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

	injectCommonHeaders(req, b.wireyVersion, b.BasicAuth)

	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("request error during join: %s", err.Error())
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("the join http request gave an unexpected status code: %d", res.StatusCode)
	}
	return nil
}

// GetPeers ...
func (b *HTTPBackend) GetPeers(ifname string) ([]Peer, error) {
	getPeersURL := fmt.Sprintf("%s/%s", b.baseurl, ifname)

	req, err := http.NewRequest("GET", getPeersURL, nil)
	if err != nil {
		return nil, err
	}

	injectCommonHeaders(req, b.wireyVersion, b.BasicAuth)

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

func injectCommonHeaders(req *http.Request, wireyVersion string, basicAuth *BasicAuth) {
	req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", httpUserAgent, wireyVersion))

	if basicAuth != nil {
		req.SetBasicAuth(basicAuth.Username, basicAuth.Password)
	}
}
