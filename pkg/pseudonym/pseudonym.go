package pseudonym

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const MinKeyLen = 32

const (
	refInfo  = "anon3anon/pseudonym/chat-ref/v1"
	sealInfo = "anon3anon/pseudonym/chat-seal/v1"
	refLen   = 16
)

var ErrKeyTooShort = fmt.Errorf("pseudonym key must be at least %d bytes", MinKeyLen)

type Keyring struct {
	refKey []byte
	aead   cipher.AEAD
}

func NewKeyring(masterKey []byte) (*Keyring, error) {
	if len(masterKey) < MinKeyLen {
		return nil, ErrKeyTooShort
	}

	refKey, err := hkdf.Key(sha256.New, masterKey, nil, refInfo, sha256.Size)
	if err != nil {
		return nil, fmt.Errorf("derive reference key: %w", err)
	}

	sealKey, err := hkdf.Key(sha256.New, masterKey, nil, sealInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive sealing key: %w", err)
	}

	block, err := aes.NewCipher(sealKey)
	if err != nil {
		return nil, fmt.Errorf("init cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init aead: %w", err)
	}

	return &Keyring{refKey: refKey, aead: aead}, nil
}

// ParseKey decodes a master key given as hex or as base64 (standard or URL
// alphabet, padded or not).
//
// Hex is tried first because the two encodings overlap: a hex string is also
// valid base64 whenever its length is a multiple of four, and decoding it as
// base64 would yield different key bytes than the operator wrote down. The
// reverse collision needs a base64 key made exclusively of hex digits, which no
// random key is.
func ParseKey(encoded string) ([]byte, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, errors.New("pseudonym key is empty")
	}

	if key, err := hex.DecodeString(trimmed); err == nil {
		return key, nil
	}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if key, err := enc.DecodeString(trimmed); err == nil {
			return key, nil
		}
	}

	return nil, errors.New("pseudonym key is neither valid hex nor valid base64")
}

// Ref returns the stable, one-way tag for chatID. Equal chat IDs always map to
// the same tag under the same key, which is what makes it usable as a primary
// key, and no tag can be turned back into a chat ID without the key.
func (k *Keyring) Ref(chatID int64) string {
	// Chat IDs are signed and can be negative; the round trip through uint64 is
	// a reinterpretation of the same bits, not a range conversion.
	var id [8]byte
	binary.BigEndian.PutUint64(id[:], uint64(chatID)) //nolint:gosec // bit reinterpretation

	mac := hmac.New(sha256.New, k.refKey)
	mac.Write(id[:])
	return hex.EncodeToString(mac.Sum(nil)[:refLen])
}

// Seal encrypts chatID so it can be recovered later by Open. A fresh random
// nonce is used per call, so two rows holding the same chat ID are not
// recognisable as such.
func (k *Keyring) Seal(chatID int64) ([]byte, error) {
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	var plaintext [8]byte
	binary.BigEndian.PutUint64(plaintext[:], uint64(chatID)) //nolint:gosec // bit reinterpretation

	return k.aead.Seal(nonce, nonce, plaintext[:], nil), nil
}

// Open recovers a chat ID produced by Seal. It fails on any sealed value that
// was truncated, tampered with, or written under a different key.
func (k *Keyring) Open(sealed []byte) (int64, error) {
	nonceSize := k.aead.NonceSize()
	if len(sealed) < nonceSize {
		return 0, errors.New("sealed chat id is too short")
	}

	plaintext, err := k.aead.Open(nil, sealed[:nonceSize], sealed[nonceSize:], nil)
	if err != nil {
		return 0, fmt.Errorf("open sealed chat id: %w", err)
	}
	if len(plaintext) != 8 {
		return 0, errors.New("sealed chat id has unexpected length")
	}

	return int64(binary.BigEndian.Uint64(plaintext)), nil //nolint:gosec // bit reinterpretation
}
