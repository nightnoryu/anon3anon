package pseudonym_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/pseudonym"
)

func testKey(b byte) []byte {
	return bytes.Repeat([]byte{b}, pseudonym.MinKeyLen)
}

func newKeyring(t *testing.T, b byte) *pseudonym.Keyring {
	t.Helper()
	keys, err := pseudonym.NewKeyring(testKey(b))
	require.NoError(t, err)
	return keys
}

func TestNewKeyringRejectsShortKeys(t *testing.T) {
	_, err := pseudonym.NewKeyring(bytes.Repeat([]byte{1}, pseudonym.MinKeyLen-1))
	require.ErrorIs(t, err, pseudonym.ErrKeyTooShort)
}

func TestRefIsDeterministicAndDistinct(t *testing.T) {
	keys := newKeyring(t, 'a')

	ref := keys.Ref(42)
	assert.Equal(t, ref, keys.Ref(42))
	assert.NotEqual(t, ref, keys.Ref(43))
	assert.NotEqual(t, ref, keys.Ref(-42))
}

func TestRefRevealsNothingAboutTheChatID(t *testing.T) {
	keys := newKeyring(t, 'a')

	ref := keys.Ref(1234567890)
	raw, err := hex.DecodeString(ref)
	require.NoError(t, err)
	assert.Len(t, raw, 16)
	assert.NotContains(t, ref, "1234567890")
}

func TestRefIsKeySpecific(t *testing.T) {
	assert.NotEqual(t, newKeyring(t, 'a').Ref(42), newKeyring(t, 'b').Ref(42))
}

func TestSealRoundTrip(t *testing.T) {
	keys := newKeyring(t, 'a')

	for _, chatID := range []int64{0, 1, -1, 1234567890, -9223372036854775808, 9223372036854775807} {
		sealed, err := keys.Seal(chatID)
		require.NoError(t, err)

		got, err := keys.Open(sealed)
		require.NoError(t, err)
		assert.Equal(t, chatID, got)
	}
}

func TestSealIsNotDeterministic(t *testing.T) {
	keys := newKeyring(t, 'a')

	first, err := keys.Seal(42)
	require.NoError(t, err)
	second, err := keys.Seal(42)
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "equal chat IDs must not produce equal ciphertext")
}

func TestOpenRejectsForeignAndDamagedValues(t *testing.T) {
	keys := newKeyring(t, 'a')
	other := newKeyring(t, 'b')

	sealed, err := keys.Seal(42)
	require.NoError(t, err)

	_, err = other.Open(sealed)
	require.Error(t, err)

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 0xff
	_, err = keys.Open(tampered)
	require.Error(t, err)

	_, err = keys.Open(sealed[:4])
	require.Error(t, err)
}

func TestParseKeyAcceptsBase64AndHex(t *testing.T) {
	key := testKey('a')

	for _, tc := range []struct {
		name    string
		encoded string
	}{
		{"std base64", base64.StdEncoding.EncodeToString(key)},
		{"raw base64", base64.RawStdEncoding.EncodeToString(key)},
		{"url base64", base64.URLEncoding.EncodeToString(key)},
		{"hex", hex.EncodeToString(key)},
		{"surrounded by whitespace", "  " + base64.StdEncoding.EncodeToString(key) + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := pseudonym.ParseKey(tc.encoded)
			require.NoError(t, err)
			assert.Equal(t, key, decoded)
		})
	}
}

func TestParseKeyRejectsGarbage(t *testing.T) {
	for _, encoded := range []string{"", "   ", "not a key!!"} {
		_, err := pseudonym.ParseKey(encoded)
		require.Error(t, err)
	}
}
