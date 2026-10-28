package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teslacost/teslacost/internal/middleware"
)

// A response carrying a secret shown once (a new API token) must not be kept for replay: the stored copy would hold the
// clear-text secret in the database while the token itself is stored hashed.
func TestIdempotencyDoesNotKeepResponsesThatCarryASecret(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	user, err := repo.CreateUser(ctx, "idem-secret@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}

	calls := 0
	h := Idempotency(repo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.HasSuffix(r.URL.Path, "/secret") {
			withholdFromIdempotency(r)
			writeJSON(w, http.StatusCreated, map[string]string{"token": "al_live_clear_text"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]int{"call": calls})
	}))
	send := func(path, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, user.ID))
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	stored := func() int {
		t.Helper()
		var n int
		if err := repo.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM idempotency_keys WHERE user_id::text = $1`, user.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	send("/api/auth/secret", "k-secret")
	if rec := send("/api/auth/secret", "k-secret"); rec.Header().Get("Idempotent-Replay") != "" || calls != 2 {
		t.Fatalf("a secret-bearing response must not be replayed, calls=%d replay=%q", calls, rec.Header().Get("Idempotent-Replay"))
	}
	if n := stored(); n != 0 {
		t.Fatalf("%d idempotency rows hold a secret-bearing response", n)
	}

	// Ordinary mutations are still replayed and stored.
	send("/api/vehicles/v/expenses", "k-plain")
	if rec := send("/api/vehicles/v/expenses", "k-plain"); rec.Header().Get("Idempotent-Replay") != "true" || calls != 3 {
		t.Fatalf("an ordinary response must still be replayed, calls=%d", calls)
	}
	if n := stored(); n != 1 {
		t.Fatalf("expected the ordinary response to be stored, got %d rows", n)
	}
}
