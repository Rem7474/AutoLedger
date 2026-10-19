package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/oauth"
)

// The example pair of RFC 7636 appendix B.
const (
	pkceVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	pkceChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	callbackURI   = "http://127.0.0.1:7777/callback"
)

type oauthFixture struct {
	t      *testing.T
	repo   *database.Repository
	router chi.Router
	user   *models.User
}

func newOAuthFixture(t *testing.T) *oauthFixture {
	t.Helper()
	repo := authTestRepo(t)
	user, err := repo.CreateUser(context.Background(), "oauth-user@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	h := NewOAuthHandler(repo, "https://ledger.example/")
	router := chi.NewRouter()
	router.Post("/oauth/register", h.Register)
	router.Post("/oauth/token", h.Token)
	router.Post("/oauth/revoke", h.Revoke)
	router.Get("/api/oauth/authorize", h.PreviewAuthorization)
	router.Post("/api/oauth/authorize", h.DecideAuthorization)
	return &oauthFixture{t: t, repo: repo, router: router, user: user}
}

func (f *oauthFixture) do(method, path, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, f.user.ID))
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f *oauthFixture) form(path string, values url.Values) *httptest.ResponseRecorder {
	return f.do(http.MethodPost, path, "application/x-www-form-urlencoded", values.Encode())
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("status %d, body %q: %v", rec.Code, rec.Body.String(), err)
	}
	return out
}

func (f *oauthFixture) register(name string, redirects ...string) string {
	f.t.Helper()
	body, _ := json.Marshal(map[string]any{"client_name": name, "redirect_uris": redirects, "token_endpoint_auth_method": "none"})
	rec := f.do(http.MethodPost, "/oauth/register", "application/json", string(body))
	if rec.Code != http.StatusCreated {
		f.t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	return decode[struct {
		ClientID string `json:"client_id"`
	}](f.t, rec).ClientID
}

func (f *oauthFixture) authorizeBody(clientID, decision string, override map[string]string) string {
	fields := map[string]string{
		"client_id": clientID, "redirect_uri": callbackURI, "response_type": "code", "code_challenge": pkceChallenge,
		"code_challenge_method": "S256", "state": "xyz", "decision": decision,
	}
	for k, v := range override {
		fields[k] = v
	}
	body, _ := json.Marshal(fields)
	return string(body)
}

// approve returns the authorization code the approval hands to the client.
func (f *oauthFixture) approve(clientID string) string {
	f.t.Helper()
	rec := f.do(http.MethodPost, "/api/oauth/authorize", "application/json", f.authorizeBody(clientID, "approve", nil))
	if rec.Code != http.StatusOK {
		f.t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
	}
	target, err := url.Parse(decode[struct {
		RedirectURL string `json:"redirect_url"`
	}](f.t, rec).RedirectURL)
	if err != nil {
		f.t.Fatal(err)
	}
	q := target.Query()
	if q.Get("state") != "xyz" || q.Get("iss") != "https://ledger.example" || !strings.HasPrefix(target.String(), callbackURI+"?") {
		f.t.Fatalf("the client gets its state, the issuer and its own address back: %s", target)
	}
	return q.Get("code")
}

func (f *oauthFixture) exchange(clientID, code, verifier string) *httptest.ResponseRecorder {
	return f.form("/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "client_id": {clientID},
		"redirect_uri": {callbackURI}, "code_verifier": {verifier},
	})
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (f *oauthFixture) validate(accessToken string) (string, error) {
	_, _, scope, err := f.repo.ValidateAPIToken(context.Background(), auth.HashAPIToken(accessToken))
	return scope, err
}

func TestOAuthAuthorizationCodeFlowGivesARevocableReadGrant(t *testing.T) {
	f := newOAuthFixture(t)
	clientID := f.register("Claude Desktop", callbackURI)

	preview := f.do(http.MethodGet, "/api/oauth/authorize?client_id="+clientID+"&redirect_uri="+url.QueryEscape(callbackURI)+
		"&response_type=code&code_challenge="+pkceChallenge+"&code_challenge_method=S256&resource="+url.QueryEscape("https://ledger.example/mcp"), "", "")
	if got := decode[map[string]string](t, preview); preview.Code != http.StatusOK || got["client_name"] != "Claude Desktop" || got["redirect_host"] != "127.0.0.1:7777" || got["scope"] != "read" {
		t.Fatalf("preview: %d %v", preview.Code, got)
	}

	rec := f.exchange(clientID, f.approve(clientID), pkceVerifier)
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("token: %d %s (cache %q)", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
	granted := decode[tokens](t, rec)
	if granted.TokenType != "Bearer" || granted.Scope != "read" || granted.ExpiresIn != 3600 || !auth.IsAPIToken(granted.AccessToken) || !strings.HasPrefix(granted.RefreshToken, oauth.RefreshTokenPrefix) {
		t.Fatalf("response: %+v", granted)
	}
	if scope, err := f.validate(granted.AccessToken); err != nil || scope != auth.ScopeRead {
		t.Fatalf("the access token is a read token of the user: %q, %v", scope, err)
	}

	list, err := f.repo.ListAPITokens(context.Background(), f.user.ID)
	if err != nil || len(list) != 1 || !list[0].OAuth || list[0].Name != "Claude Desktop" || list[0].Scope != auth.ScopeRead {
		t.Fatalf("the grant is listed with the client's name: %+v, %v", list, err)
	}
	if list[0].ExpiresAt == nil || time.Until(*list[0].ExpiresAt) < 80*24*time.Hour {
		t.Errorf("the listed expiry is the refresh token's, not the hour of the access token: %v", list[0].ExpiresAt)
	}

	// Renewing replaces both tokens.
	renewed := f.form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {granted.RefreshToken}})
	if renewed.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", renewed.Code, renewed.Body.String())
	}
	second := decode[tokens](t, renewed)
	if second.AccessToken == granted.AccessToken || second.RefreshToken == granted.RefreshToken || second.Scope != "read" {
		t.Fatalf("a renewal issues new tokens: %+v", second)
	}
	if _, err := f.validate(granted.AccessToken); err == nil {
		t.Error("the access token before the renewal must stop working")
	}
	if _, err := f.validate(second.AccessToken); err != nil {
		t.Errorf("the renewed access token works: %v", err)
	}

	// Presenting the refresh token that was already renewed ends the grant, for whoever holds the newer one too.
	if rec := f.form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {granted.RefreshToken}}); rec.Code != http.StatusBadRequest || decode[oauthError](t, rec).Error != "invalid_grant" {
		t.Fatalf("a reused refresh token is refused: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := f.validate(second.AccessToken); err == nil {
		t.Error("reuse revokes the grant, the newest access token included")
	}
	if rec := f.form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {second.RefreshToken}}); rec.Code != http.StatusBadRequest {
		t.Errorf("nor can the newest refresh token be used: %d", rec.Code)
	}
}

func TestOAuthCodeWorksOnceAndOnlyForTheRequestThatAskedForIt(t *testing.T) {
	f := newOAuthFixture(t)
	clientID := f.register("App", callbackURI)
	otherID := f.register("Other app", callbackURI)

	code := f.approve(clientID)
	if rec := f.exchange(clientID, code, pkceVerifier); rec.Code != http.StatusOK {
		t.Fatalf("first exchange: %d %s", rec.Code, rec.Body.String())
	}
	if rec := f.exchange(clientID, code, pkceVerifier); rec.Code != http.StatusBadRequest || decode[oauthError](t, rec).Error != "invalid_grant" {
		t.Errorf("a code cannot be exchanged twice: %d %s", rec.Code, rec.Body.String())
	}

	for name, try := range map[string]func(code string) *httptest.ResponseRecorder{
		"another verifier": func(c string) *httptest.ResponseRecorder {
			return f.exchange(clientID, c, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		},
		"another client": func(c string) *httptest.ResponseRecorder { return f.exchange(otherID, c, pkceVerifier) },
		"another redirect": func(c string) *httptest.ResponseRecorder {
			return f.form("/oauth/token", url.Values{"grant_type": {"authorization_code"}, "code": {c}, "client_id": {clientID},
				"redirect_uri": {"http://127.0.0.1:7777/elsewhere"}, "code_verifier": {pkceVerifier}})
		},
	} {
		code := f.approve(clientID)
		if rec := try(code); rec.Code != http.StatusBadRequest || decode[oauthError](t, rec).Error != "invalid_grant" {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
		if rec := f.exchange(clientID, code, pkceVerifier); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: a refused attempt spends the code, the right one cannot follow: %d", name, rec.Code)
		}
	}

	expired := "expired-code"
	if err := f.repo.CreateOAuthCode(context.Background(), oauth.Hash(expired),
		models.OAuthCode{ClientID: clientID, UserID: f.user.ID, RedirectURI: callbackURI, CodeChallenge: pkceChallenge, Scope: "read"}, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if rec := f.exchange(clientID, expired, pkceVerifier); rec.Code != http.StatusBadRequest {
		t.Errorf("an expired code is refused: %d", rec.Code)
	}

	if rec := f.form("/oauth/token", url.Values{"grant_type": {"password"}}); decode[oauthError](t, rec).Error != "unsupported_grant_type" {
		t.Errorf("only the code and refresh grants exist: %s", rec.Body.String())
	}
	if rec := f.form("/oauth/token", url.Values{"grant_type": {"authorization_code"}, "code": {code}}); decode[oauthError](t, rec).Error != "invalid_request" {
		t.Errorf("missing parameters: %s", rec.Body.String())
	}
}

func TestOAuthAuthorizationRequestIsCheckedAgainstTheRegisteredClient(t *testing.T) {
	f := newOAuthFixture(t)
	clientID := f.register("App", callbackURI)

	for name, c := range map[string]struct {
		override map[string]string
		code     string
	}{
		"unknown client":          {map[string]string{"client_id": "al_client_unknown"}, "oauth.unknown_client"},
		"unregistered return":     {map[string]string{"redirect_uri": "https://evil.example/cb"}, "oauth.invalid_redirect_uri"},
		"no PKCE":                 {map[string]string{"code_challenge": "", "code_challenge_method": ""}, "oauth.invalid_request"},
		"plain PKCE":              {map[string]string{"code_challenge_method": "plain"}, "oauth.invalid_request"},
		"implicit grant":          {map[string]string{"response_type": "token"}, "oauth.invalid_request"},
		"write scope":             {map[string]string{"scope": "read write"}, "oauth.invalid_scope"},
		"token for another place": {map[string]string{"resource": "https://other.example/mcp"}, "oauth.invalid_resource"},
		"unknown decision":        {map[string]string{"decision": "maybe"}, "oauth.invalid_request"},
	} {
		rec := f.do(http.MethodPost, "/api/oauth/authorize", "application/json", f.authorizeBody(clientID, "approve", c.override))
		if got := decode[map[string]any](t, rec); rec.Code != http.StatusBadRequest || got["code"] != c.code {
			t.Errorf("%s: %d %v, want %s", name, rec.Code, got, c.code)
		}
		if strings.Contains(rec.Body.String(), "redirect_url") {
			t.Errorf("%s: a refused request is never redirected", name)
		}
	}

	// A return address on another port is the loopback case, accepted.
	if rec := f.do(http.MethodPost, "/api/oauth/authorize", "application/json", f.authorizeBody(clientID, "deny", map[string]string{"redirect_uri": "http://127.0.0.1:50999/callback"})); rec.Code != http.StatusOK {
		t.Errorf("loopback port: %d %s", rec.Code, rec.Body.String())
	}

	deny := f.do(http.MethodPost, "/api/oauth/authorize", "application/json", f.authorizeBody(clientID, "deny", nil))
	target, _ := url.Parse(decode[map[string]string](t, deny)["redirect_url"])
	if target.Query().Get("error") != "access_denied" || target.Query().Get("code") != "" || target.Query().Get("state") != "xyz" {
		t.Errorf("a refusal tells the client so, without a code: %s", target)
	}
}

func TestOAuthRegistrationRules(t *testing.T) {
	f := newOAuthFixture(t)
	register := func(body string) *httptest.ResponseRecorder {
		return f.do(http.MethodPost, "/oauth/register", "application/json", body)
	}
	for name, body := range map[string]string{
		"no redirect":   `{"client_name":"x","redirect_uris":[]}`,
		"plain http":    `{"client_name":"x","redirect_uris":["http://example.org/cb"]}`,
		"script scheme": `{"client_name":"x","redirect_uris":["javascript:alert(1)"]}`,
		"wildcard":      `{"client_name":"x","redirect_uris":["https://*.example.org/cb"]}`,
		"too many":      `{"client_name":"x","redirect_uris":["https://a.example/1","https://a.example/2","https://a.example/3","https://a.example/4","https://a.example/5","https://a.example/6","https://a.example/7","https://a.example/8","https://a.example/9","https://a.example/10","https://a.example/11"]}`,
		"not json":      `nope`,
	} {
		if rec := register(body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	if rec := register(`{"redirect_uris":["https://a.example/cb"],"token_endpoint_auth_method":"client_secret_basic"}`); rec.Code != http.StatusBadRequest || decode[oauthError](t, rec).Error != "invalid_client_metadata" {
		t.Errorf("a client with a secret is not supported: %d %s", rec.Code, rec.Body.String())
	}
	if rec := register(`{"redirect_uris":["https://a.example/cb"],"grant_types":["implicit"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("only the code grant: %d", rec.Code)
	}

	rec := register(`{"client_name":"  Evil\u0000 ‮` + strings.Repeat("n", 200) + `","redirect_uris":["https://a.example/cb"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	got := decode[models.OAuthClient](t, rec)
	if strings.ContainsRune(got.Name, 0) || len([]rune(got.Name)) > oauth.MaxClientNameLength {
		t.Errorf("the name is cleaned and bounded: %q", got.Name)
	}
	if rec := register(`{"redirect_uris":["https://a.example/cb"]}`); decode[models.OAuthClient](t, rec).Name != "Unnamed application" {
		t.Error("a client without a name is shown as unnamed")
	}

	var existing int
	if err := f.repo.Pool().QueryRow(context.Background(), `SELECT COUNT(*) FROM oauth_clients`).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	client := func(id string) *models.OAuthClient {
		return &models.OAuthClient{ID: id, Name: "n", RedirectURIs: []string{"https://a.example/cb"}}
	}
	if err := f.repo.CreateOAuthClient(context.Background(), client("al_client_last"), existing+1); err != nil {
		t.Fatalf("the instance accepts clients up to its cap: %v", err)
	}
	if err := f.repo.CreateOAuthClient(context.Background(), client("al_client_over"), existing+1); err != database.ErrTooManyOAuthClients {
		t.Errorf("past the cap registration is refused: %v", err)
	}
}

func TestOAuthAuthorizingAgainReplacesTheGrantAndRevocationEndsIt(t *testing.T) {
	f := newOAuthFixture(t)
	clientID := f.register("App", callbackURI)
	otherID := f.register("Other app", callbackURI)

	first := decode[tokens](t, f.exchange(clientID, f.approve(clientID), pkceVerifier))
	second := decode[tokens](t, f.exchange(clientID, f.approve(clientID), pkceVerifier))
	if _, err := f.validate(first.AccessToken); err == nil {
		t.Error("authorizing again replaces the earlier grant of the client")
	}
	if list, _ := f.repo.ListAPITokens(context.Background(), f.user.ID); len(list) != 1 {
		t.Errorf("one grant per client and user, got %d", len(list))
	}

	// Another client cannot revoke it; the client itself can, with either token.
	f.form("/oauth/revoke", url.Values{"token": {second.RefreshToken}, "client_id": {otherID}})
	if _, err := f.validate(second.AccessToken); err != nil {
		t.Fatal("a client revokes only its own grants")
	}
	if rec := f.form("/oauth/revoke", url.Values{"token": {second.AccessToken}, "client_id": {clientID}}); rec.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := f.validate(second.AccessToken); err == nil {
		t.Error("a revoked grant stops working")
	}
	if rec := f.form("/oauth/revoke", url.Values{"token": {"never-issued"}, "client_id": {clientID}}); rec.Code != http.StatusOK {
		t.Errorf("revoking an unknown token is not an error: %d", rec.Code)
	}
	if rec := f.form("/oauth/revoke", url.Values{"token": {"x"}}); rec.Code != http.StatusBadRequest {
		t.Errorf("a revocation names its client: %d", rec.Code)
	}
}

func TestOAuthExpiredAccessTokenStopsWorkingButTheGrantRenews(t *testing.T) {
	f := newOAuthFixture(t)
	clientID := f.register("App", callbackURI)
	granted := decode[tokens](t, f.exchange(clientID, f.approve(clientID), pkceVerifier))

	pool := f.repo.Pool()
	if _, err := pool.Exec(context.Background(), `UPDATE api_tokens SET expires_at = NOW() - INTERVAL '1 minute' WHERE oauth_client_id = $1`, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.validate(granted.AccessToken); err == nil {
		t.Fatal("an expired access token is refused")
	}
	if rec := f.form("/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {granted.RefreshToken}}); rec.Code != http.StatusOK {
		t.Fatalf("the refresh token renews it: %d %s", rec.Code, rec.Body.String())
	}

	if _, err := pool.Exec(context.Background(), `UPDATE api_tokens SET refresh_expires_at = NOW() - INTERVAL '1 minute' WHERE oauth_client_id = $1`, clientID); err != nil {
		t.Fatal(err)
	}
	if n, err := f.repo.CleanupOAuth(context.Background()); err != nil || n != 1 {
		t.Errorf("a grant whose refresh token expired is purged: %d, %v", n, err)
	}
}

func TestOAuthMetadataAdvertisesOnlyWhatIsSupported(t *testing.T) {
	h := NewOAuthHandler(nil, "https://ledger.example/")
	rec := httptest.NewRecorder()
	h.Metadata(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	got := decode[map[string]any](t, rec)

	if got["issuer"] != "https://ledger.example" || got["token_endpoint"] != "https://ledger.example/oauth/token" ||
		got["registration_endpoint"] != "https://ledger.example/oauth/register" || got["authorization_endpoint"] != "https://ledger.example/oauth/authorize" {
		t.Errorf("endpoints hang off the configured address, without a double slash: %v", got)
	}
	for key, want := range map[string]string{
		"code_challenge_methods_supported":      "S256",
		"token_endpoint_auth_methods_supported": "none",
		"response_types_supported":              "code",
		"scopes_supported":                      "read",
	} {
		list, _ := got[key].([]any)
		if len(list) != 1 || list[0] != want {
			t.Errorf("%s = %v, want only %q", key, got[key], want)
		}
	}
	if h.Resource() != "https://ledger.example/mcp" {
		t.Errorf("resource %q", h.Resource())
	}
}
