package wireguard

const confTemplate = `[Interface]
ListenPort = {{ .Interface.ListenPort  }}
PrivateKey = {{ .Interface.PrivateKey }}
{{ range .Peers }}

[Peer]
AllowedIPs = {{ .AllowedIPs }}
Endpoint = {{ .Endpoint }}
PublicKey = {{ .PublicKey }}
{{ end }}`
