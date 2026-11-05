//go:build !fastbcrypt

package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Guards the work factor of the hashes a production build writes (the fastbcrypt tag lowers it for CI test runs).
func TestPasswordHashesUseCostTwelve(t *testing.T) {
	hash, err := HashPassword("SecurePassword123!")
	if err != nil {
		t.Fatal(err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil || cost != 12 {
		t.Fatalf("cost = %d, err = %v; want 12", cost, err)
	}
}
