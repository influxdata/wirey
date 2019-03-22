package utils

import (
	"crypto/sha256"
	"fmt"
	"net"
)

// PublicKeySHA256 tuns bytes into SHA256
func PublicKeySHA256(key []byte) string {
	h := sha256.New()
	h.Write(key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetIPv4ForInterfaceName returns interface ip from interface name
func GetIPv4ForInterfaceName(ifname string) (ifaceip net.IP) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == ifname {
			if addrs, err := inter.Addrs(); err == nil {
				for _, addr := range addrs {
					switch ip := addr.(type) {
					case *net.IPNet:
						if ip.IP.DefaultMask() != nil {
							return (ip.IP)
						}
					}
				}
			}
		}
	}
	return (nil)
}
