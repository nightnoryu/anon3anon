package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const byteLen = 8

func New() (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
