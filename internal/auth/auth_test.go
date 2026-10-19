package auth

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecurePassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Fatalf("Password verification failed for valid password")
	}

	if CheckPassword("WrongPassword", hash) {
		t.Fatalf("Password verification succeeded for invalid password")
	}

	_, err = HashPassword("short")
	if err != ErrPasswordTooShort {
		t.Fatalf("Expected ErrPasswordTooShort, got %v", err)
	}
}

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "secret-jwt-key"
	userID := "user-uuid-12345"
	email := "user@example.com"

	token, err := GenerateToken(userID, email, secret, 24)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("Token claims mismatch: got user %s, email %s", claims.UserID, claims.Email)
	}

	// Validate with wrong secret
	_, err = ValidateToken(token, "wrong-secret")
	if err == nil {
		t.Fatalf("Token validation should have failed with wrong secret")
	}
}

func TestGenerateAccessToken(t *testing.T) {
	secret := "secret-jwt-key"
	userID := "user-uuid-12345"
	email := "user@example.com"

	token, err := GenerateAccessToken(userID, email, secret, 15)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("Access token claims mismatch: got user %s, email %s", claims.UserID, claims.Email)
	}
}

func TestRefreshTokenGenerationAndHashing(t *testing.T) {
	plainToken1, hash1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	if len(plainToken1) != 64 { // 32 bytes in hex = 64 characters
		t.Fatalf("Expected 64 hex characters, got %d", len(plainToken1))
	}

	if len(hash1) != 64 { // SHA-256 in hex = 64 characters
		t.Fatalf("Expected 64 hex characters for SHA-256 hash, got %d", len(hash1))
	}

	// Verify hashing is deterministic
	recalculatedHash := HashRefreshToken(plainToken1)
	if recalculatedHash != hash1 {
		t.Fatalf("Hash mismatch: expected %s, got %s", hash1, recalculatedHash)
	}

	// Verify randomness (two generated tokens must differ)
	plainToken2, hash2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed to generate second refresh token: %v", err)
	}
	if plainToken1 == plainToken2 || hash1 == hash2 {
		t.Fatalf("Two randomly generated tokens should not collide")
	}
}

func TestNewUUID(t *testing.T) {
	u1, err := NewUUID()
	if err != nil {
		t.Fatalf("Failed to generate UUID: %v", err)
	}
	if len(u1) != 36 {
		t.Fatalf("Expected 36 chars UUID, got %s (len %d)", u1, len(u1))
	}

	u2, err := NewUUID()
	if err != nil {
		t.Fatalf("Failed to generate second UUID: %v", err)
	}
	if u1 == u2 {
		t.Fatalf("UUID collision detected: %s == %s", u1, u2)
	}
}
