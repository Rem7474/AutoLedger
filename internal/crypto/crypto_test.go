package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
)

// legacyEncrypt builds a value as the previous version stored it: one SHA-256 of the secret as the key.
func legacyEncrypt(t *testing.T, e *Encryptor, plain string) string {
	t.Helper()
	gcm, err := newGCM(e.legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil))
}

func newTestEncryptor(t *testing.T, secret string) *Encryptor {
	t.Helper()
	e, err := NewEncryptor(secret)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	return e
}

func TestEncryptDecrypt(t *testing.T) {
	enc := newTestEncryptor(t, "test-secret-key-12345")
	secretToken := "tm_api_secret_token_abcdef1234567890"

	encrypted, err := enc.Encrypt(secretToken)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	if !strings.HasPrefix(encrypted, "v2:") || strings.Contains(encrypted, secretToken) {
		t.Fatalf("expected a v2 value that hides the plain text, got %q", encrypted)
	}
	if IsLegacy(encrypted) {
		t.Fatal("a v2 value is not legacy")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil || decrypted != secretToken {
		t.Fatalf("Decrypt() = %q, %v; want %q", decrypted, err, secretToken)
	}
}

func TestEmptyValues(t *testing.T) {
	enc := newTestEncryptor(t, "key")
	for name, f := range map[string]func(string) (string, error){"Encrypt": enc.Encrypt, "Decrypt": enc.Decrypt, "Rotate": enc.Rotate} {
		if out, err := f(""); err != nil || out != "" {
			t.Fatalf("%s of an empty value = %q, %v; want empty", name, out, err)
		}
	}
	if IsLegacy("") {
		t.Fatal("an empty value needs no rotation")
	}
	if _, err := NewEncryptor(""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("NewEncryptor(\"\") error = %v, want ErrEmptyKey", err)
	}
}

func TestSameValueEncryptsDifferently(t *testing.T) {
	enc := newTestEncryptor(t, "k")
	a, _ := enc.Encrypt("same")
	b, _ := enc.Encrypt("same")
	if a == b {
		t.Fatal("two encryptions of one value must differ (random salt and nonce)")
	}
}

func TestLegacyValueIsStillRead(t *testing.T) {
	enc := newTestEncryptor(t, "test-secret-key-12345")
	old := legacyEncrypt(t, enc, "legacy-token")
	if !IsLegacy(old) {
		t.Fatal("a legacy value must be reported as legacy")
	}
	if got, err := enc.Decrypt(old); err != nil || got != "legacy-token" {
		t.Fatalf("Decrypt(legacy) = %q, %v", got, err)
	}
}

func TestRotateMovesLegacyToV2AndIsIdempotent(t *testing.T) {
	enc := newTestEncryptor(t, "test-secret-key-12345")
	rotated, err := enc.Rotate(legacyEncrypt(t, enc, "legacy-token"))
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if IsLegacy(rotated) {
		t.Fatalf("rotated value is still legacy: %q", rotated)
	}
	if got, err := enc.Decrypt(rotated); err != nil || got != "legacy-token" {
		t.Fatalf("Decrypt(rotated) = %q, %v", got, err)
	}
	again, err := enc.Rotate(rotated)
	if err != nil || again != rotated {
		t.Fatalf("rotating a v2 value must leave it unchanged, got %q, %v", again, err)
	}
}

func TestRotateRefusesAValueOfAnotherKey(t *testing.T) {
	other := newTestEncryptor(t, "another-secret")
	if _, err := newTestEncryptor(t, "this-secret").Rotate(legacyEncrypt(t, other, "x")); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("Rotate with the wrong key error = %v, want ErrInvalidCiphertext", err)
	}
}

func TestWrongKeyAndTamperedValuesAreRefused(t *testing.T) {
	enc := newTestEncryptor(t, "right-key")
	sealed, _ := enc.Encrypt("secret")

	if _, err := newTestEncryptor(t, "wrong-key").Decrypt(sealed); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("wrong key error = %v, want ErrInvalidCiphertext", err)
	}

	raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, "v2:"))
	raw[len(raw)-1] ^= 0xff
	if _, err := enc.Decrypt("v2:" + base64.StdEncoding.EncodeToString(raw)); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("tampered value error = %v, want ErrInvalidCiphertext", err)
	}

	for name, bad := range map[string]string{"not base64": "v2:***", "too short": "v2:" + base64.StdEncoding.EncodeToString([]byte("short")), "no nonce": "v2:" + base64.StdEncoding.EncodeToString(make([]byte, saltSize))} {
		if _, err := enc.Decrypt(bad); !errors.Is(err, ErrInvalidCiphertext) {
			t.Fatalf("%s: error = %v, want ErrInvalidCiphertext", name, err)
		}
	}
	if _, err := enc.Decrypt("AAAA"); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("a legacy value too short error = %v, want ErrInvalidCiphertext", err)
	}
	if _, err := enc.Decrypt("***"); err == nil {
		t.Fatal("a value that is not base64 must be refused")
	}
}

func TestKeyCacheDoesNotChangeResults(t *testing.T) {
	enc := newTestEncryptor(t, "cache-key")
	sealed, _ := enc.Encrypt("value")
	for i := 0; i < 3; i++ {
		if got, err := enc.Decrypt(sealed); err != nil || got != "value" {
			t.Fatalf("Decrypt #%d = %q, %v", i, got, err)
		}
	}
	// Past its size the cache starts over; every value stays readable.
	enc.cacheSize = 2
	var values []string
	for i := 0; i < 5; i++ {
		v, err := enc.Encrypt("v")
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, v)
	}
	for _, v := range values {
		if got, err := enc.Decrypt(v); err != nil || got != "v" {
			t.Fatalf("Decrypt after the cache turned over = %q, %v", got, err)
		}
	}
}
