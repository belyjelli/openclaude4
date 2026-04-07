package session

import (
	"crypto/rand"
	"encoding/hex"
)

// NewRandomID returns a 16-character hex session id.
func NewRandomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "local"
	}
	return hex.EncodeToString(b)
}
