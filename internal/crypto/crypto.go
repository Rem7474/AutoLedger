package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCiphertext = errors.New("invalid or corrupted ciphertext")
	ErrEmptyKey          = errors.New("encryption key cannot be empty")
)

// A stored value is either the legacy form, base64(nonce || sealed) under a key made of one SHA-256 of the secret,
// or the current form, "v2:" + base64(salt || nonce || sealed) under a key derived from the secret and a random
// salt with Argon2id. Base64 never contains ':', so the prefix tells the two apart.
const (
	v2Prefix = "v2:"
	// purpose binds the derived key to this use: the same secret used elsewhere derives another key, and a value
	// sealed for another purpose does not open here.
	purpose = "autoledger/secrets-at-rest/v2"

	saltSize = 16
	// Argon2id cost (the OWASP minimum: 2 passes over 19 MiB, one lane), light enough for a small home server. A
	// derivation takes tens of milliseconds, which is the point against an offline guess of a weak secret; derived
	// keys are cached by salt.
	argonTime    = 2
	argonMemory  = 19 * 1024
	argonThreads = 1
	keyCacheSize = 256
)

// Encryptor handles AES-256-GCM encryption and decryption.
type Encryptor struct {
	secret    []byte
	legacyKey []byte

	mu        sync.Mutex
	cache     map[[saltSize]byte][]byte
	cacheSize int
}

// NewEncryptor creates an Encryptor from the application secret. New values are written in the v2 form; the
// legacy form is still read, and Rotate rewrites it.
func NewEncryptor(secretKey string) (*Encryptor, error) {
	if secretKey == "" {
		return nil, ErrEmptyKey
	}
	legacy := sha256.Sum256([]byte(secretKey))
	return &Encryptor{
		secret:    append([]byte(purpose+"\x00"), secretKey...),
		legacyKey: legacy[:],
		cache:     make(map[[saltSize]byte][]byte),
		cacheSize: keyCacheSize,
	}, nil
}

// IsLegacy reports whether a stored value is in the legacy form and should be rotated.
func IsLegacy(stored string) bool {
	return stored != "" && !strings.HasPrefix(stored, v2Prefix)
}

// keyFor derives the key of a salt, memoised: a sync decrypts the same few credentials every cycle.
func (e *Encryptor) keyFor(salt []byte) []byte {
	var id [saltSize]byte
	copy(id[:], salt)
	e.mu.Lock()
	defer e.mu.Unlock()
	if key, ok := e.cache[id]; ok {
		return key
	}
	key := argon2.IDKey(e.secret, salt, argonTime, argonMemory, argonThreads, 32)
	if len(e.cache) >= e.cacheSize {
		e.cache = make(map[[saltSize]byte][]byte)
	}
	e.cache[id] = key
	return key
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Encrypt seals plain text in the v2 form.
func (e *Encryptor) Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	gcm, err := newGCM(e.keyFor(salt))
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// The version and the purpose are authenticated, not only the payload.
	sealed := gcm.Seal(nil, nonce, []byte(plainText), []byte(v2Prefix+purpose))
	out := make([]byte, 0, len(salt)+len(nonce)+len(sealed))
	out = append(append(append(out, salt...), nonce...), sealed...)
	return v2Prefix + base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt opens a stored value, in either form.
func (e *Encryptor) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if rest, ok := strings.CutPrefix(stored, v2Prefix); ok {
		return e.decryptV2(rest)
	}
	return e.decryptLegacy(stored)
}

func (e *Encryptor) decryptV2(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	if len(data) < saltSize {
		return "", ErrInvalidCiphertext
	}
	salt, rest := data[:saltSize], data[saltSize:]
	gcm, err := newGCM(e.keyFor(salt))
	if err != nil {
		return "", err
	}
	if len(rest) < gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}
	nonce, sealed := rest[:gcm.NonceSize()], rest[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, []byte(v2Prefix+purpose))
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plain), nil
}

func (e *Encryptor) decryptLegacy(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(e.legacyKey)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}
	plain, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plain), nil
}

// Rotate rewrites a legacy value in the v2 form; a value already in v2, or empty, comes back unchanged.
func (e *Encryptor) Rotate(stored string) (string, error) {
	if !IsLegacy(stored) {
		return stored, nil
	}
	plain, err := e.decryptLegacy(stored)
	if err != nil {
		return "", err
	}
	return e.Encrypt(plain)
}
