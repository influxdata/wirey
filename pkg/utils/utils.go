package utils

import (
	"crypto/sha256"
	"fmt"
)

// PublicKeySHA256 tuns bytes into SHA256
func PublicKeySHA256(key []byte) string {
	h := sha256.New()
	h.Write(key)
	return fmt.Sprintf("%x", h.Sum(nil))
}
