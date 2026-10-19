// Package oauth holds the rules of the authorization server that lets assistants (MCP clients) obtain a read token:
// which redirect addresses are acceptable, the PKCE check and the secrets it hands out. The endpoints are in the
// handlers package.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/auth"
)

const (
	// AccessTokenTTL is how long an access token works before the client renews it with its refresh token.
	AccessTokenTTL = time.Hour
	// RefreshTokenTTL is how long a grant lives without being renewed; every renewal starts it again.
	RefreshTokenTTL = 90 * 24 * time.Hour
	// CodeTTL is how long an authorization code can be exchanged.
	CodeTTL = time.Minute

	// RefreshTokenPrefix tells a refresh token from an access token (auth.TokenPrefix).
	RefreshTokenPrefix = "al_refresh_"
	clientIDPrefix     = "al_client_"

	// MaxRedirectURIs and MaxRedirectURILength bound what a client may register.
	MaxRedirectURIs      = 10
	MaxRedirectURILength = 500
	// MaxClientNameLength is the length of the name shown on the consent page and in the token list.
	MaxClientNameLength = 100
)

// Scopes are what a grant may do. Only reading exists.
var Scopes = []string{auth.ScopeRead}

// NormalizeScope returns the scope a request asks for: read when none is given, and false for anything else.
func NormalizeScope(raw string) (string, bool) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return auth.ScopeRead, true
	}
	for _, f := range fields {
		if f != auth.ScopeRead {
			return "", false
		}
	}
	return auth.ScopeRead, true
}

// forbiddenSchemes can run script or reach local content from a redirect, so no client may register them.
var forbiddenSchemes = map[string]bool{
	"javascript": true, "data": true, "file": true, "vbscript": true, "about": true, "blob": true,
	"ftp": true, "ws": true, "wss": true, "chrome": true, "view-source": true,
}

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

func isLoopback(u *url.URL) bool {
	switch strings.ToLower(u.Hostname()) {
	case "127.0.0.1", "::1", "localhost":
		return true
	}
	return false
}

// ValidRedirectURI reports whether a client may register uri: HTTPS, http on a loopback address (a native app picks
// a free port at run time), or the private scheme of a desktop app. Fragments, credentials and wildcards are refused.
func ValidRedirectURI(raw string) bool {
	if raw == "" || len(raw) > MaxRedirectURILength || strings.ContainsAny(raw, "#*") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Scheme == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	switch {
	case scheme == "https":
		return u.Hostname() != ""
	case scheme == "http":
		return isLoopback(u)
	case forbiddenSchemes[scheme] || !schemePattern.MatchString(scheme):
		return false
	default:
		return true
	}
}

// MatchRedirectURI reports whether requested is one of the registered addresses. An address matches exactly; a
// loopback http address also matches with another port.
func MatchRedirectURI(registered []string, requested string) bool {
	for _, r := range registered {
		if r == requested {
			return true
		}
	}
	req, err := url.Parse(requested)
	if err != nil || req.Scheme != "http" || !isLoopback(req) {
		return false
	}
	for _, r := range registered {
		reg, err := url.Parse(r)
		if err != nil || reg.Scheme != "http" || !isLoopback(reg) {
			continue
		}
		if strings.EqualFold(reg.Hostname(), req.Hostname()) && reg.EscapedPath() == req.EscapedPath() && reg.RawQuery == req.RawQuery {
			return true
		}
	}
	return false
}

// RedirectHost names where an approval sends the user back, for the consent page: the host of a web address, the
// scheme of an application one.
func RedirectHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return u.Host
	}
	return u.Scheme + "://"
}

var verifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
var challengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// ValidChallenge reports whether challenge has the shape of an S256 code challenge (a base64url SHA-256).
func ValidChallenge(challenge string) bool { return challengePattern.MatchString(challenge) }

// VerifyPKCE checks the code verifier a client presents against the S256 challenge it sent when asking for the code.
func VerifyPKCE(verifier, challenge string) bool {
	if !verifierPattern.MatchString(verifier) || !ValidChallenge(challenge) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// Hash is the SHA-256 form in which codes and refresh tokens are stored.
func Hash(secret string) string { return auth.HashAPIToken(secret) }

func random(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// NewClientID returns the identifier of a registering client.
func NewClientID() (string, error) {
	s, err := random(16)
	return clientIDPrefix + s, err
}

// NewCode returns an authorization code and the hash that is stored.
func NewCode() (code, hash string, err error) {
	code, err = random(32)
	return code, Hash(code), err
}

// NewRefreshToken returns a refresh token and the hash that is stored.
func NewRefreshToken() (token, hash string, err error) {
	s, err := random(32)
	token = RefreshTokenPrefix + s
	return token, Hash(token), err
}
