package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

// registerMCPRoutes registers the Model Context Protocol endpoint and the metadata that points a client to the
// authorization server. The endpoint checks its own bearer token (read scope, API tokens only) because a refused
// request must answer with the WWW-Authenticate header MCP clients start the authorization from. It exists only when
// the instance enables MCP.
func (h *apiHandlers) registerMCPRoutes(r chi.Router) {
	if h.mcp == nil {
		return
	}
	r.Get("/.well-known/oauth-protected-resource", h.mcp.ResourceMetadata().ServeHTTP)
	r.Get("/.well-known/oauth-protected-resource/mcp", h.mcp.ResourceMetadata().ServeHTTP)
	// The server is stateless: there is no stream to open with a GET and no session to close with a DELETE. They are
	// still routed to the endpoint, so a client that probes with one learns where to authorize (401 with the
	// challenge) or that the method is not supported (405), instead of receiving the web app's page.
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodHead, http.MethodDelete} {
		r.With(mcpLimiters(mcpRequestsPerAddress, mcpRequestsPerToken)...).Method(method, "/mcp", h.mcp.Endpoint())
	}
}

// Limits of the MCP endpoint per minute. A cloud connector reaches an instance from a few shared addresses, so the
// address limit is only a coarse bound on unauthenticated traffic; the limit that matters to a session is the token's.
const (
	mcpRequestsPerAddress = 600
	mcpRequestsPerToken   = 120
)

// mcpLimiters bounds the requests of an address, then those of one access token.
func mcpLimiters(perAddress, perToken int) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		httprate.LimitByIP(perAddress, time.Minute),
		httprate.Limit(perToken, time.Minute, httprate.WithKeyFuncs(mcpTokenKey)),
	}
}

// mcpTokenKey identifies a bearer token by its hash, so the credential is never held by the limiter. A request
// without one is keyed by its address.
func mcpTokenKey(r *http.Request) (string, error) {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && token != "" {
		sum := sha256.Sum256([]byte(token))
		return "token:" + hex.EncodeToString(sum[:]), nil
	}
	return "address:" + r.RemoteAddr, nil
}
