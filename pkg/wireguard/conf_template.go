package wireguard

const confTemplate = `[Interface]
ListenPort = {{ .Interface.ListenPort }}
PrivateKey = {{ .Interface.PrivateKey }}

{{ range .Peers }}
[Peer]
PublicKey = {{ .PublicKey }}
AllowedIPs = {{ .AllowedIPs }}
Endpoint = {{ .Endpoint }}
{{ end }}`
