package wireguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderConfiguration(t *testing.T) {
	conf := Configuration{
		Interface: Interface{
			ListenPort: 49082,
			PrivateKey: "iOIMgrmMHt/L/GT+Fw2DruosUXDlBgSclXo52S//41k=",
		},
		Peers: []Peer{
			{
				PublicKey:  "Rg3XQfzH0LWuUBy/MHZxMcCLxiMaE5BS1hY/pncQ0G4=",
				AllowedIPs: "10.0.0.1/32",
				Endpoint:   "172.31.23.163:50113",
			},
			{
				PublicKey:  "nAMY8gSy32B7rLV8kiLq4GKJBbYT3amT+c0DI5vikik=",
				AllowedIPs: "10.0.0.2/32",
				Endpoint:   "172.31.23.162:43043",
			},
		},
	}
	rendered, err := RenderConfiguration(conf)

	if err != nil {
		t.Error(err)
	}

	expected := `[Interface]
ListenPort = 49082
PrivateKey = iOIMgrmMHt/L/GT+Fw2DruosUXDlBgSclXo52S//41k=


[Peer]
PublicKey = Rg3XQfzH0LWuUBy/MHZxMcCLxiMaE5BS1hY/pncQ0G4=
AllowedIPs = 10.0.0.1/32
Endpoint = 172.31.23.163:50113


[Peer]
PublicKey = nAMY8gSy32B7rLV8kiLq4GKJBbYT3amT+c0DI5vikik=
AllowedIPs = 10.0.0.2/32
Endpoint = 172.31.23.162:43043
`

	assert.Equal(t, expected, string(rendered))
}
