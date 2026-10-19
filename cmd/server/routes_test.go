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
	registerAPIRoutes(r, testJWTSecret, fakeTokenValidator{scope: scope}, fakeSessionChecker{}, passThrough, &apiHandlers{})
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

	session, err := auth.GenerateAccessToken("user-1", "user@example.com", testJWTSecret, 5)
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
		"POST /api/auth/login",
		"POST /api/auth/logout",
		"POST /api/auth/refresh",
		"POST /api/auth/register",
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

// fakeSessionChecker lets every session through: these tests are about which routes need which credential.
type fakeSessionChecker struct{}

func (fakeSessionChecker) IsSessionActive(context.Context, string, string) (bool, error) {
	return true, nil
}
