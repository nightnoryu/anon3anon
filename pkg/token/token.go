package token

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/go-faster/errors"
)

const byteLen = 8

func New() (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "read random bytes")
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
