package token_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/token"
)

func TestNew(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for range 1000 {
		tok, err := token.New()
		require.NoError(t, err)
		assert.NotEmpty(t, tok)

		_, err = base64.RawURLEncoding.DecodeString(tok)
		require.NoError(t, err, "token must be valid raw-url base64")

		_, dup := seen[tok]
		assert.False(t, dup, "tokens must not collide")
		seen[tok] = struct{}{}
	}
}
