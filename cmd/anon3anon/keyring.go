package main

import (
	"fmt"
	"strings"

	"anon3anon/pkg/pseudonym"
)

func initKeyring(encodedKey string) (*pseudonym.Keyring, error) {
	if strings.TrimSpace(encodedKey) == "" {
		return nil, fmt.Errorf(
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
