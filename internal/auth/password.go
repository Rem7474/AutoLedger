package auth

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

// passwordCost is the bcrypt work factor of every hash this process generates. Hashes carry their own cost, so a
// lower one (see cost_fast.go) never prevents the check of an existing hash.
var passwordCost = 12

// HashPassword generates a bcrypt hash of the password with cost 12.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), passwordCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a plaintext password with a bcrypt hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MaxPasswordBytes is the longest password bcrypt takes into account: it silently ignores what follows, so a longer
// one would give a false sense of strength.
const MaxPasswordBytes = 72

// ErrPasswordTooLong is returned for a password bcrypt would truncate.
var ErrPasswordTooLong = errors.New("password must be at most 72 bytes")

// dummyHash is compared against when no account matches, so that an unknown address takes as long to refuse as a
// wrong password and the response time does not reveal which addresses have an account. It hashes random bytes
// drawn at start-up: nothing can match it, and there is no literal password in the source.
var dummyHash = func() []byte {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}
	h, err := bcrypt.GenerateFromPassword(secret, passwordCost)
	if err != nil {
		panic(err)
	}
	return h
}()

// CheckPasswordAgainstNobody spends the time of a password check without any account.
func CheckPasswordAgainstNobody(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
