package syncnode

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateNodeToken returns a random token for agents.
func GenerateNodeToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
