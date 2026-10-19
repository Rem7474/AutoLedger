package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

// registerOAuthPublicRoutes registers the authorization server an assistant talks to: its metadata, the registration
// of a client, and the token and revocation endpoints. None carries a session, the credential is the code and its
// PKCE verifier or the refresh token. They exist only when the instance enables MCP.
func (h *apiHandlers) registerOAuthPublicRoutes(r chi.Router) {
	if h.oauth == nil {
		return
	}
	r.Get("/.well-known/oauth-authorization-server", h.oauth.Metadata)
	r.Get("/.well-known/oauth-authorization-server/mcp", h.oauth.Metadata)
	// Registering needs no account: a few registrations per hour and address are plenty for a person setting up clients.
	r.With(httprate.LimitByIP(10, time.Hour)).Post("/oauth/register", h.oauth.Register)
	r.With(httprate.LimitByIP(60, time.Minute)).Post("/oauth/token", h.oauth.Token)
	r.With(httprate.LimitByIP(60, time.Minute)).Post("/oauth/revoke", h.oauth.Revoke)
}

// registerOAuthSessionRoutes registers the two calls the consent page makes for the signed-in user. The page itself
// is the front end's /oauth/authorize route, which the server leaves to the single-page app. Approving creates a
// credential, so it needs an active session like creating an API token does.
func (h *apiHandlers) registerOAuthSessionRoutes(r chi.Router, activeSession func(http.Handler) http.Handler) {
	if h.oauth == nil {
		return
	}
	r.Get("/api/oauth/authorize", h.oauth.PreviewAuthorization)
	r.With(httprate.LimitByIP(20, time.Minute), activeSession).Post("/api/oauth/authorize", h.oauth.DecideAuthorization)
}
