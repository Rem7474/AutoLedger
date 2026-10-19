package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teslacost/teslacost/internal/auth"
)

func TestAuthenticateJWTAcceptsBearerHeader(t *testing.T) {
	secret := "test-secret"
	token, err := auth.GenerateAccessToken("user-1", "user@example.com", secret, 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var gotUserID string
	handler := AuthenticateJWT(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotUserID != "user-1" {
		t.Errorf("expected user-1, got %q", gotUserID)
	}
}

func TestAuthenticateJWTFallsBackToAccessTokenCookie(t *testing.T) {
	secret := "test-secret"
	token, err := auth.GenerateAccessToken("user-2", "cookie@example.com", secret, 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var gotUserID string
	handler := AuthenticateJWT(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotUserID != "user-2" {
		t.Errorf("expected user-2, got %q", gotUserID)
	}
}

func TestAuthenticateJWTPrefersHeaderOverCookie(t *testing.T) {
	secret := "test-secret"
	headerToken, _ := auth.GenerateAccessToken("header-user", "h@example.com", secret, 15)
	cookieToken, _ := auth.GenerateAccessToken("cookie-user", "c@example.com", secret, 15)

	var gotUserID string
	handler := AuthenticateJWT(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	req.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: cookieToken})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotUserID != "header-user" {
		t.Errorf("expected header token to take precedence, got %q", gotUserID)
	}
}

func TestAuthenticateJWTRejectsMissingCredentials(t *testing.T) {
	handler := AuthenticateJWT("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without credentials")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

type fakeTokenValidator map[string]string // token hash -> user ID

func (f fakeTokenValidator) ValidateAPIToken(_ context.Context, tokenHash string) (string, string, string, error) {
	if userID, ok := f[tokenHash]; ok {
		return userID, userID + "@example.com", auth.ScopeIngest, nil
	}
	return "", "", "", errors.New("unknown token")
}

// scopedTokenValidator knows tokens by hash with the scope each was created with.
type scopedTokenValidator map[string]string // token hash -> scope

func (f scopedTokenValidator) ValidateAPIToken(_ context.Context, tokenHash string) (string, string, string, error) {
	if scope, ok := f[tokenHash]; ok {
		return "user-1", "user-1@example.com", scope, nil
	}
	return "", "", "", errors.New("unknown token")
}

func TestRequireTokenScope(t *testing.T) {
	secret := "test-secret"
	ingest := auth.TokenPrefix + "ingest-token-0123456789abcdef0123456789abcdef"
	read := auth.TokenPrefix + "read-token-0123456789abcdef0123456789abcdef0123"
	validator := scopedTokenValidator{auth.HashAPIToken(ingest): auth.ScopeIngest, auth.HashAPIToken(read): auth.ScopeRead}
	session, err := auth.GenerateAccessToken("user-2", "user2@example.com", secret, 15)
	if err != nil {
		t.Fatal(err)
	}

	status := func(scopes []string, bearer string) int {
		handler := AuthenticateIntegration(secret, validator)(RequireTokenScope(scopes...)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+bearer)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	cases := []struct {
		name   string
		scopes []string
		bearer string
		want   int
	}{
		{"an ingest token pushes", []string{auth.ScopeIngest}, ingest, http.StatusOK},
		{"a read token cannot push", []string{auth.ScopeIngest}, read, http.StatusForbidden},
		{"a read token reads", []string{auth.ScopeIngest, auth.ScopeRead}, read, http.StatusOK},
		{"an ingest token reads what it always read", []string{auth.ScopeIngest, auth.ScopeRead}, ingest, http.StatusOK},
		{"an ingest token cannot reach a read-only route", []string{auth.ScopeRead}, ingest, http.StatusForbidden},
		{"a session is not scoped", []string{auth.ScopeIngest}, session, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := status(tc.scopes, tc.bearer); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAPITokensOnlyOpenIntegrationRoutes(t *testing.T) {
	secret := "test-secret"
	apiToken := auth.TokenPrefix + "0123456789abcdef0123456789abcdef0123456789abcdef"
	validator := fakeTokenValidator{auth.HashAPIToken(apiToken): "user-1"}
	session, err := auth.GenerateAccessToken("user-2", "user2@example.com", secret, 15)
	if err != nil {
		t.Fatal(err)
	}

	call := func(mw func(http.Handler) http.Handler, bearer string) (int, string) {
		var gotUserID string
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUserID = GetUserID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+bearer)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code, gotUserID
	}

	if code, _ := call(AuthenticateJWT(secret), apiToken); code != http.StatusUnauthorized {
		t.Errorf("session routes: a valid API token got %d, want 401", code)
	}
	if code, user := call(AuthenticateJWT(secret), session); code != http.StatusOK || user != "user-2" {
		t.Errorf("session routes: a session got %d for %q, want 200 for user-2", code, user)
	}
	if code, user := call(AuthenticateIntegration(secret, validator), apiToken); code != http.StatusOK || user != "user-1" {
		t.Errorf("integration routes: a valid API token got %d for %q, want 200 for user-1", code, user)
	}
	if code, _ := call(AuthenticateIntegration(secret, validator), auth.TokenPrefix+"revoked"); code != http.StatusUnauthorized {
		t.Errorf("integration routes: an unknown API token got %d, want 401", code)
	}
	if code, user := call(AuthenticateIntegration(secret, validator), session); code != http.StatusOK || user != "user-2" {
		t.Errorf("integration routes: a session got %d for %q, want 200 for user-2", code, user)
	}
}
