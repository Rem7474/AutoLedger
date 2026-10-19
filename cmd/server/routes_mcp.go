package main

import (
	"net/http"
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
	limit := httprate.LimitByIP(120, time.Minute)
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodHead, http.MethodDelete} {
		r.With(limit).Method(method, "/mcp", h.mcp.Endpoint())
	}
}
