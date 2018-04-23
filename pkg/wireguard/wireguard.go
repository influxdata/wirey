package wireguard

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

type Interface struct {
	ListenPort int
	PrivateKey string
}

type Peer struct {
	PublicKey  string
	AllowedIPs string
	Endpoint   string
}

type Configuration struct {
	Interface Interface
	Peers     []Peer
}

const (
	errorWiregurdNotFound = "the wireguard (wg) command is not available in your PATH"
)

func wg(stdin io.Reader, arg ...string) ([]byte, error) {
	path, err := exec.LookPath("wg")
	if err != nil {
		return nil, fmt.Errorf(errorWiregurdNotFound)
	}

	cmd := exec.Command(path, arg...)

	return cmd.Output()
}

func Genkey() ([]byte, error) {
	return wg(nil, "genkey")
}

func ExtractPubKey(privateKey []byte) ([]byte, error) {
	stdin := bytes.NewReader(privateKey)
	return wg(stdin, "pubkey")
}

func SetConf(ifname string, conf Configuration) ([]byte, error) {
	cfile, err := ioutil.TempFile("", "wgconfig")
	if err != nil {
		return nil, err
	}
	defer os.Remove(cfile.Name())
	rendered, err := RenderConfiguration(conf)
	if err != nil {
		return nil, err
	}
	if _, err := cfile.Write(rendered); err != nil {
		return nil, err
	}
	return wg(nil, "setconf", "wg0", cfile.Name())
}

func RenderConfiguration(conf Configuration) ([]byte, error) {
	t := template.Must(template.New("config").Parse(confTemplate))
	buf := &bytes.Buffer{}

	err := t.Execute(buf, conf)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
