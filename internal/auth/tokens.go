package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const TokenPrefix = "al_live_"

// What an API token may do. Ingest is the original behaviour (push events and read the Home Assistant views);
// Read only reads; Write reads and also records what the MCP write tools offer (grouping drives into a carpool). Only
// an assistant's grant (OAuth) can hold Write: it is not a scope a token created from the account page can have. A
// session is not scoped.
const (
	ScopeIngest = "ingest"
	ScopeRead   = "read"
	ScopeWrite  = "write"
)

// CanRead reports whether a token of this scope may read the integration views.
func CanRead(scope string) bool { return scope == ScopeRead || scope == ScopeWrite }

// ValidScope reports whether scope names a scope a token can be created with from the account page.
func ValidScope(scope string) bool {
	return scope == ScopeIngest || scope == ScopeRead
}

// GenerateAPIToken creates a cryptographically secure random token, its display prefix, and its SHA-256 hash.
func GenerateAPIToken() (rawToken, prefix, tokenHash string, err error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomPart := hex.EncodeToString(bytes)
	rawToken = TokenPrefix + randomPart
	prefix = rawToken[:len(TokenPrefix)+4] + "..."
	tokenHash = HashAPIToken(rawToken)
	return rawToken, prefix, tokenHash, nil
}

// HashAPIToken computes the SHA-256 hexadecimal hash of an API token.
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// IsAPIToken checks if a token string matches the API token prefix convention.
func IsAPIToken(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), TokenPrefix)
}
