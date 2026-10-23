package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/config"
	"github.com/teslacost/teslacost/internal/handlers"
)

const testJWTSecret = "route-table-test-secret-0123456789abcdef"

// fakeTokenValidator accepts any API token, with the scope it names (ingest when empty).
type fakeTokenValidator struct{ scope string }

func (f fakeTokenValidator) ValidateAPIToken(_ context.Context, _ string) (string, string, string, error) {
	scope := f.scope
	if scope == "" {
		scope = auth.ScopeIngest
	}
	return "user-1", "user@example.com", scope, nil
}

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// The messages the authentication middleware answers with; any other response came from the route's handler.
const (
	noCredentialsMessage = "missing Authorization header"
	sessionOnlyMessage   = "only accepted on integration endpoints"
	badSessionMessage    = "Invalid or expired token"
)

func newRouteTable(t *testing.T) chi.Router { return newRouteTableFor(t, "") }

// newRouteTableFor builds the route table with API tokens that carry the given scope.
func newRouteTableFor(t *testing.T, scope string) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	passThrough := func(next http.Handler) http.Handler { return next }
	registerSystemRoutes(r, nil)
	registerAPIRoutes(r, testJWTSecret, fakeTokenValidator{scope: scope}, fakeSessionChecker{}, passThrough, &apiHandlers{
		oauth: handlers.NewOAuthHandler(nil, "http://localhost"),
		mcp:   handlers.NewMCPHandler(nil, fakeTokenValidator{scope: scope}, "http://localhost", "test"),
	})
	return r
}

func probe(r chi.Router, method, pattern, bearer string) string {
	path := routeParam.ReplaceAllString(pattern, "1")
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	func() {
		defer func() { _ = recover() }() // the handlers are nil: reaching one panics, which is what "let through" means here
		r.ServeHTTP(rec, req)
	}()
	return rec.Body.String()
}

// TestRouteTableAuthTiers pins which routes are public, which an API token opens and which need a session, so
// that a route moved into the wrong group fails here instead of widening access.
func TestRouteTableAuthTiers(t *testing.T) {
	r := newRouteTable(t)

	session, err := auth.GenerateSessionAccessToken("user-1", "user@example.com", "session-1", testJWTSecret, 5)
	if err != nil {
		t.Fatal(err)
	}
	apiToken := auth.TokenPrefix + "route-table-test"

	var public, integration, sessionOnly []string
	err = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		label := method + " " + route
		if strings.HasSuffix(route, "/*") {
			return nil
		}
		switch {
		case !strings.Contains(probe(r, method, route, ""), noCredentialsMessage):
			public = append(public, label)
		case !strings.Contains(probe(r, method, route, apiToken), sessionOnlyMessage):
			integration = append(integration, label)
			assertSessionOpens(t, r, method, route, session)
		default:
			sessionOnly = append(sessionOnly, label)
			assertSessionOpens(t, r, method, route, session)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPublic := []string{
		"GET /api/auth/config",
		"GET /api/auth/oidc/callback",
		"GET /api/auth/oidc/login",
		"GET /api/health",
		"GET /api/version",
		"GET /healthz",
		"GET /.well-known/oauth-authorization-server",
		"GET /.well-known/oauth-authorization-server/mcp",
		"GET /.well-known/oauth-protected-resource",
		"GET /.well-known/oauth-protected-resource/mcp",
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"POST /api/auth/refresh",
		"POST /api/auth/register",
		"POST /mcp", // checks its own bearer token (read scope) so that a refusal carries the WWW-Authenticate header
		"GET /mcp",
		"HEAD /mcp",
		"DELETE /mcp",
		"POST /oauth/register",
		"POST /oauth/revoke",
		"POST /oauth/token",
	}
	wantIntegration := []string{
		"GET /api/integrations/homeassistant/vehicles",
		"GET /api/integrations/homeassistant/vehicles/{vehicleId}",
		"GET /api/integrations/homeassistant/vehicles/{vehicleId}/metrics",
		"POST /api/integrations/homeassistant/event",
		"GET /api/integrations/v1/vehicles",
		"GET /api/integrations/v1/vehicles/{vehicleId}",
		"GET /api/integrations/v1/vehicles/{vehicleId}/costs",
		"GET /api/integrations/v1/vehicles/{vehicleId}/data-gaps",
		"GET /api/integrations/v1/vehicles/{vehicleId}/trips",
		"GET /api/integrations/v1/vehicles/{vehicleId}/monthly-costs",
		"GET /api/integrations/v1/vehicles/{vehicleId}/reminders",
		"GET /api/integrations/v1/vehicles/{vehicleId}/ownership",
		"GET /api/integrations/v1/vehicles/{vehicleId}/energy-efficiency",
		"GET /api/integrations/v1/vehicles/{vehicleId}/expenses",
		"GET /api/integrations/v1/vehicles/{vehicleId}/tags",
		"GET /api/integrations/v1/vehicles/{vehicleId}/carpool",
		"GET /api/integrations/v1/vehicles/{vehicleId}/charging-places",
		"GET /api/integrations/v1/vehicles/{vehicleId}/trip-groups",
		"GET /api/integrations/v1/vehicles/{vehicleId}/tires",
		"GET /api/integrations/v1/vehicles/{vehicleId}/odometer",
		"GET /api/integrations/v1/vehicles/{vehicleId}/data-sources",
		"GET /api/integrations/v1/comparisons",
		"GET /api/integrations/v1/fleet",
	}
	assertSameRoutes(t, "public", public, wantPublic)
	assertSameRoutes(t, "API token", integration, wantIntegration)

	if len(sessionOnly) < 150 {
		t.Errorf("only %d session routes found, the route table lost routes", len(sessionOnly))
	}
	for _, label := range sessionOnly {
		if strings.Contains(label, "/api/integrations/") {
			t.Errorf("%s needs a session although it is an integration route", label)
		}
	}
}

// TestReadTokenCannotPushEvents pins the scope of each integration route: a read token reads the Home Assistant
// views and the read API and is refused on the route that writes, an ingest token keeps the push and the Home
// Assistant views but is refused on the read API.
func TestReadTokenCannotPushEvents(t *testing.T) {
	apiToken := auth.TokenPrefix + "route-table-test"
	const scopeRefusal = "does not have the scope"
	const push = "/api/integrations/homeassistant/event"
	const read = "/api/integrations/homeassistant/vehicles"
	const readAPI = "/api/integrations/v1/vehicles"

	readOnly := newRouteTableFor(t, auth.ScopeRead)
	if body := probe(readOnly, http.MethodPost, push, apiToken); !strings.Contains(body, scopeRefusal) {
		t.Errorf("a read token must be refused on %s, got %q", push, body)
	}
	if body := probe(readOnly, http.MethodGet, read, apiToken); strings.Contains(body, scopeRefusal) {
		t.Errorf("a read token must reach %s, got %q", read, body)
	}

	if body := probe(readOnly, http.MethodGet, readAPI, apiToken); strings.Contains(body, scopeRefusal) {
		t.Errorf("a read token must reach %s, got %q", readAPI, body)
	}

	ingest := newRouteTableFor(t, auth.ScopeIngest)
	if body := probe(ingest, http.MethodGet, readAPI, apiToken); !strings.Contains(body, scopeRefusal) {
		t.Errorf("an ingest token must be refused on %s, got %q", readAPI, body)
	}
	for _, route := range []struct{ method, path string }{{http.MethodPost, push}, {http.MethodGet, read}} {
		if body := probe(ingest, route.method, route.path, apiToken); strings.Contains(body, scopeRefusal) {
			t.Errorf("an ingest token must reach %s %s, got %q", route.method, route.path, body)
		}
	}
}

// TestEveryIntegrationRouteChecksTheTokenScope fails for an integration route that forgot RequireTokenScope: a token
// whose scope the route does not list must be refused on every one of them.
func TestEveryIntegrationRouteChecksTheTokenScope(t *testing.T) {
	r := newRouteTableFor(t, "no-such-scope")
	apiToken := auth.TokenPrefix + "route-table-test"
	var checked int
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/api/integrations/") {
			return nil
		}
		checked++
		if body := probe(r, method, route, apiToken); !strings.Contains(body, "does not have the scope") {
			t.Errorf("%s %s accepts a token of any scope, wrap it in RequireTokenScope", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no integration route found")
	}
}

func assertSessionOpens(t *testing.T, r chi.Router, method, route, session string) {
	t.Helper()
	body := probe(r, method, route, session)
	if strings.Contains(body, noCredentialsMessage) || strings.Contains(body, badSessionMessage) {
		t.Errorf("%s %s: a session must open the route, got %s", method, route, body)
	}
}

func assertSameRoutes(t *testing.T, tier string, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s routes differ\n got: %v\nwant: %v", tier, got, want)
	}
}

// endedSessionChecker answers that every session was revoked.
type endedSessionChecker struct{}

func (endedSessionChecker) IsSessionActive(context.Context, string, string) (bool, error) {
	return false, nil
}

// TestEveryRouteRefusesARevokedSession fails for a route that honours an access token whose session was ended: every
// route a session opens, on the session tier and on the integration tier, answers 401 to it, while the integration
// routes keep accepting an API token, which is not bound to a session.
func TestEveryRouteRefusesARevokedSession(t *testing.T) {
	r := chi.NewRouter()
	passThrough := func(next http.Handler) http.Handler { return next }
	registerAPIRoutes(r, testJWTSecret, fakeTokenValidator{}, endedSessionChecker{}, passThrough, &apiHandlers{
		oauth: handlers.NewOAuthHandler(nil, "http://localhost"),
		mcp:   handlers.NewMCPHandler(nil, fakeTokenValidator{}, "http://localhost", "test"),
	})
	session, err := auth.GenerateSessionAccessToken("user-1", "user@example.com", "session-1", testJWTSecret, 5)
	if err != nil {
		t.Fatal(err)
	}
	apiToken := auth.TokenPrefix + "route-table-test"

	var refused, tokenRoutes int
	err = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasSuffix(route, "/*") || !strings.Contains(probe(r, method, route, ""), noCredentialsMessage) {
			return nil // a public route
		}
		refused++
		if body := probe(r, method, route, session); !strings.Contains(body, "Session is no longer active") {
			t.Errorf("%s %s honours the access token of a revoked session: %q", method, route, body)
		}
		if strings.HasPrefix(route, "/api/integrations/") {
			tokenRoutes++
			if body := probe(r, method, route, apiToken); strings.Contains(body, "Session is no longer active") {
				t.Errorf("%s %s asks an API token for a session: %q", method, route, body)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if refused < 150 || tokenRoutes == 0 {
		t.Fatalf("only %d routes checked (%d with an API token), the route table lost routes", refused, tokenRoutes)
	}
}

// fakeSessionChecker lets every session through: these tests are about which routes need which credential.
type fakeSessionChecker struct{}

func (fakeSessionChecker) IsSessionActive(context.Context, string, string) (bool, error) {
	return true, nil
}

// TestMCPEndpointNeverFallsThroughToTheWebApp pins what a client sees when it probes the endpoint with a method the
// stateless server does not use: the authorization challenge without a token, a 405 with one, and never the
// single-page app's HTML, from which it cannot tell how the server wants to be authorized.
func TestMCPEndpointNeverFallsThroughToTheWebApp(t *testing.T) {
	r := newRouteTableFor(t, auth.ScopeRead)
	registerSPA(r)

	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodHead, http.MethodDelete} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(method, "/mcp", nil))
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("WWW-Authenticate"), "/.well-known/oauth-protected-resource/mcp") {
			t.Errorf("%s /mcp without a token: status %d, challenge %q", method, rec.Code, rec.Header().Get("WWW-Authenticate"))
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		req := httptest.NewRequest(method, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer "+auth.TokenPrefix+"route-table-test")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed || strings.Contains(rec.Body.String(), "<html") {
			t.Errorf("%s /mcp with a token: status %d, want 405 from the endpoint", method, rec.Code)
		}
	}
}

func TestUnknownDiscoveryDocumentIsNotTheWebApp(t *testing.T) {
	r := newRouteTable(t)
	registerSPA(r)
	for _, path := range []string{"/.well-known/openid-configuration", "/.well-known/anything-else"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "<html") {
			t.Errorf("%s: status %d, want a plain 404", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("the OAuth metadata must still be served: status %d", rec.Code)
	}
}

// TestMCPEndpointAnswersABrowserClientsProbe covers a client that runs in a browser (the sign-in of a web connector):
// its requests carry an Origin header, and it must still get the 401 challenge and a CORS grant, not a 403 from the
// origin check, while the application's own routes keep refusing a foreign origin.
func TestMCPEndpointAnswersABrowserClientsProbe(t *testing.T) {
	cfg := &config.Config{AppBaseURL: "https://ledger.example", MCPEnabled: true}
	r := newRouter(cfg, nil)
	registerAPIRoutes(r, testJWTSecret, fakeTokenValidator{scope: auth.ScopeRead}, fakeSessionChecker{}, func(next http.Handler) http.Handler { return next },
		&apiHandlers{oauth: handlers.NewOAuthHandler(nil, cfg.AppBaseURL), mcp: handlers.NewMCPHandler(nil, fakeTokenValidator{scope: auth.ScopeRead}, cfg.AppBaseURL, "test")})

	send := func(method, path string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "https://ledger.example"+path, nil)
		req.Header.Set("Origin", "https://claude.ai")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	rec := send(http.MethodPost, "/mcp", nil)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("WWW-Authenticate"), "resource_metadata=") ||
		rec.Header().Get("Access-Control-Allow-Origin") != "*" || !strings.Contains(rec.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate") ||
		rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("POST /mcp from another origin (a wildcard grant never comes with credentials): status %d, headers %v", rec.Code, rec.Header())
	}
	if rec := send(http.MethodOptions, "/mcp", map[string]string{"Access-Control-Request-Method": "POST"}); rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("preflight on /mcp: status %d, headers %v", rec.Code, rec.Header())
	}
	if rec := send(http.MethodGet, "/.well-known/oauth-authorization-server", nil); rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("the authorization server metadata must be readable from a page: status %d, headers %v", rec.Code, rec.Header())
	}
	if rec := send(http.MethodPost, "/api/auth/login", nil); rec.Code != http.StatusForbidden {
		t.Errorf("the application's own routes still refuse a foreign origin: status %d", rec.Code)
	}
}
