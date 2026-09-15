package main

import (
	"strings"

	"github.com/go-faster/errors"

	"anon3anon/pkg/pseudonym"
)

func initKeyring(encodedKey string) (*pseudonym.Keyring, error) {
	if strings.TrimSpace(encodedKey) == "" {
		return nil, errors.Errorf(
			"%s_PSEUDONYM_KEY is required: generate one with `openssl rand -base64 %d` and keep it "+
				"in your secret store, not on the data volume",
			strings.ToUpper(appID), pseudonym.MinKeyLen,
		)
	}

	key, err := pseudonym.ParseKey(encodedKey)
	if err != nil {
		return nil, err
	}
	return pseudonym.NewKeyring(key)
}
