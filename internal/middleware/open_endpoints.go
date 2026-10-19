package middleware

import (
	"context"
	"net/http"
)

type openEndpointKey struct{}

// OpenEndpoints makes the given paths callable from any origin, for the endpoints a browser-based client talks to
// without the user's session: the OAuth metadata, registration, token and revocation endpoints and the MCP endpoint.
// They carry no ambient credential (no cookie is read: the credential is a bearer token, an authorization code with its
// PKCE verifier, or nothing), so a page of another site gains nothing by calling them that it could not get by calling
// them from its own server. Put it before OriginCheck and the CORS handler: it answers the preflight itself and marks
// the request so OriginCheck lets it through, which is what lets the endpoint answer with its own 401 challenge
// instead of a 403 the client cannot interpret.
func OpenEndpoints(paths ...string) func(http.Handler) http.Handler {
	open := make(map[string]bool, len(paths))
	for _, p := range paths {
		open[p] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !open[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Add("Vary", "Origin")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, MCP-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
				h.Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h.Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), openEndpointKey{}, true)))
		})
	}
}

// SkipOnOpenEndpoints applies mw to every request except those OpenEndpoints marked: the application's CORS handler
// grants credentials to the origins it trusts, which must not be added to an open endpoint's wildcard.
func SkipOnOpenEndpoints(mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isOpenEndpoint(r) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}

func isOpenEndpoint(r *http.Request) bool {
	open, _ := r.Context().Value(openEndpointKey{}).(bool)
	return open
}

// OpenEndpointPaths lists the paths of the MCP endpoint and of the OAuth endpoints its clients use.
func OpenEndpointPaths() []string {
	return []string{
		"/mcp",
		"/oauth/register", "/oauth/token", "/oauth/revoke",
		"/.well-known/oauth-authorization-server", "/.well-known/oauth-authorization-server/mcp",
		"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp",
	}
}
