package wireguard

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"
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

	cmd.Stdin = stdin
	var buf bytes.Buffer
	cmd.Stderr = &buf
	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("%s - %s", err.Error(), buf.String())
	}
	return output, nil

}

func Genkey() ([]byte, error) {
	result, err := wg(nil, "genkey")
	if err != nil {
		return nil, fmt.Errorf("error generating the private key for wireguard: %s", err.Error())
	}
	return result, nil
}

func ExtractPubKey(privateKey []byte) ([]byte, error) {
	stdin := bytes.NewReader(privateKey)
	result, err := wg(stdin, "pubkey")
	if err != nil {
		return nil, fmt.Errorf("error extracting the public key: %s", err.Error())
	}
	return result, nil
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

	result, err := wg(nil, "setconf", "wg0", cfile.Name())

	if err != nil {
		return nil, fmt.Errorf("error setting the configuration for wireguard: %s", err.Error())
	}
	return result, nil
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
