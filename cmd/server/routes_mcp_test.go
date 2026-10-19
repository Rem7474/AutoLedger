package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPLimitIsPerAccessTokenSoOneSessionDoesNotStarveAnother(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	var chain http.Handler = h
	limiters := mcpLimiters(100, 2)
	for i := len(limiters) - 1; i >= 0; i-- {
		chain = limiters[i](chain)
	}
	call := func(token string) int {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "203.0.113.7:5555" // the same cloud connector address for every session
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, req)
		return rec.Code
	}

	if call("al_live_a") != http.StatusOK || call("al_live_a") != http.StatusOK {
		t.Fatal("a token may send its allowance")
	}
	if got := call("al_live_a"); got != http.StatusTooManyRequests {
		t.Errorf("a token over its allowance: status %d, want 429", got)
	}
	if got := call("al_live_b"); got != http.StatusOK {
		t.Errorf("another token behind the same address keeps its own allowance: status %d", got)
	}
}

func TestMCPLimitFallsBackToTheAddressWithoutAToken(t *testing.T) {
	for _, header := range []string{"", "Bearer ", "Basic abc"} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.RemoteAddr = "198.51.100.1:1"
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		key, err := mcpTokenKey(req)
		if err != nil || key != "address:198.51.100.1:1" {
			t.Errorf("%q: key %q, err %v", header, key, err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer al_live_secret")
	if key, _ := mcpTokenKey(req); key == "" || key == "token:al_live_secret" {
		t.Errorf("the credential must not be the key itself: %q", key)
	}
}
