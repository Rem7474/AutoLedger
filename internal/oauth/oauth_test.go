package oauth

import (
	"strings"
	"testing"
)

func TestValidRedirectURI(t *testing.T) {
	tooLong := "https://example.org/" + strings.Repeat("a", MaxRedirectURILength)
	cases := map[string]bool{
		"https://claude.ai/api/mcp/auth_callback":            true,
		"http://127.0.0.1:53682/callback":                    true,
		"http://localhost/callback":                          true,
		"http://[::1]:8000/cb":                               true,
		"cursor://anysphere.cursor-retrieval/oauth/callback": true,
		"vscode://ms-vscode.mcp/callback":                    true,
		"http://example.org/callback":                        false, // plain http is for loopback only
		"https://":                                           false,
		"https://example.org/cb#fragment":                    false,
		"https://*.example.org/cb":                           false,
		"https://user:pass@example.org/cb":                   false,
		"javascript:alert(1)":                                false,
		"data:text/html,x":                                   false,
		"file:///etc/passwd":                                 false,
		"/relative/path":                                     false,
		"":                                                   false,
		tooLong:                                              false,
	}
	for uri, want := range cases {
		if got := ValidRedirectURI(uri); got != want {
			t.Errorf("ValidRedirectURI(%q) = %v, want %v", uri, got, want)
		}
	}
}

func TestMatchRedirectURI(t *testing.T) {
	registered := []string{"http://127.0.0.1:3000/callback", "https://claude.ai/cb", "cursor://anysphere.cursor-retrieval/cb"}
	cases := map[string]bool{
		"https://claude.ai/cb":                           true,
		"cursor://anysphere.cursor-retrieval/cb":         true,
		"http://127.0.0.1:3000/callback":                 true,
		"http://127.0.0.1:61234/callback":                true, // a native app picks a free port
		"http://127.0.0.1:61234/other":                   false,
		"http://127.0.0.1:61234/callback?x=1":            false,
		"http://localhost:61234/callback":                false, // another host name
		"https://claude.ai:8443/cb":                      false, // only loopback may change its port
		"https://claude.ai/cb/":                          false,
		"https://evil.example/cb":                        false,
		"https://claude.ai/cb?next=https://evil.example": false,
		"": false,
	}
	for uri, want := range cases {
		if got := MatchRedirectURI(registered, uri); got != want {
			t.Errorf("MatchRedirectURI(%q) = %v, want %v", uri, got, want)
		}
	}
	if MatchRedirectURI([]string{"https://claude.ai/cb"}, "http://127.0.0.1:1/cb") {
		t.Error("a loopback address matches only when a loopback address was registered")
	}
}

func TestVerifyPKCE(t *testing.T) {
	// The example of RFC 7636 appendix B.
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if !VerifyPKCE(verifier, challenge) {
		t.Fatal("the RFC verifier must match its challenge")
	}
	for name, v := range map[string]string{
		"another verifier": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"too short":        "short",
		"bad characters":   strings.Repeat("a", 42) + "!",
		"empty":            "",
	} {
		if VerifyPKCE(v, challenge) {
			t.Errorf("%s must not verify", name)
		}
	}
	if VerifyPKCE(verifier, verifier) {
		t.Error("a plain challenge (the verifier itself) is not accepted")
	}
}

func TestNormalizeScope(t *testing.T) {
	for raw, want := range map[string]string{"": "read", "read": "read", " read ": "read", "read read": "read", "write": "write", "read write": "write", "write read": "write"} {
		scope, ok := NormalizeScope(raw)
		if !ok || scope != want {
			t.Errorf("NormalizeScope(%q) = %q, %v, want %q", raw, scope, ok, want)
		}
	}
	for _, raw := range []string{"ingest", "read ingest", "admin", "WRITE"} {
		if scope, ok := NormalizeScope(raw); ok {
			t.Errorf("NormalizeScope(%q) = %q, want a refusal", raw, scope)
		}
	}
}

func TestSecretsAreDistinctAndStoredHashed(t *testing.T) {
	code, codeHash, err := NewCode()
	if err != nil || code == "" || codeHash == code || Hash(code) != codeHash {
		t.Fatalf("code %q hash %q err %v", code, codeHash, err)
	}
	refresh, refreshHash, err := NewRefreshToken()
	if err != nil || !strings.HasPrefix(refresh, RefreshTokenPrefix) || Hash(refresh) != refreshHash {
		t.Fatalf("refresh %q hash %q err %v", refresh, refreshHash, err)
	}
	other, _, _ := NewRefreshToken()
	if other == refresh {
		t.Error("two refresh tokens must differ")
	}
	id, _ := NewClientID()
	if !strings.HasPrefix(id, "al_client_") {
		t.Errorf("client id %q", id)
	}
}

func TestRedirectHost(t *testing.T) {
	for uri, want := range map[string]string{
		"https://claude.ai/api/cb": "claude.ai",
		"http://127.0.0.1:5000/cb": "127.0.0.1:5000",
		"cursor://anysphere/cb":    "cursor://",
	} {
		if got := RedirectHost(uri); got != want {
			t.Errorf("RedirectHost(%q) = %q, want %q", uri, got, want)
		}
	}
}
