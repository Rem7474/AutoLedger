package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/teslacost/teslacost/internal/auth"
)

type contextKey string

const (
	UserIDKey    contextKey = "userID"
	UserEmailKey contextKey = "userEmail"
	// SessionIDKey holds the session an access token was issued for; absent for an API token or an older token.
	SessionIDKey contextKey = "sessionID"
	// TokenScopeKey holds the scope of the API token that authenticated the request; absent for a session.
	TokenScopeKey contextKey = "tokenScope"
)

// accessTokenCookieName mirrors the constant of the same name in the handlers package
// (an internal/middleware -> internal/handlers import would be circular).
const accessTokenCookieName = "teslacost_access_token"

// extractBearerToken reads the JWT from the Authorization header (API/programmatic clients)
// or, failing that, from the HttpOnly access-token cookie set by the browser-facing SPA.
func extractBearerToken(r *http.Request) (string, error) {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", fmt.Errorf("invalid Authorization format, expected 'Bearer <token>'")
		}
		return parts[1], nil
	}

	if cookie, err := r.Cookie(accessTokenCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	return "", fmt.Errorf("missing Authorization header or %s cookie", accessTokenCookieName)
}

// SessionChecker tells whether a session (refresh token family) of a user can still renew its access.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}

// RequireActiveSession refuses a request whose access token comes from a session that was revoked or has expired,
// and a token that names no session: an access token is honoured only while the session it was issued for is
// active, so ending a session (logout, "sign out everywhere", a revoked device) cuts off its access token at once
// instead of leaving it valid until it expires. Run it after AuthenticateJWT or AuthenticateIntegration. A request
// authenticated by an API token (TokenScopeKey set) has no session and passes: its token is checked by the
// validator. The 401 makes the web app renew its session, and the retry carries a token bound to the new one.
func RequireActiveSession(checker SessionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, isAPIToken := r.Context().Value(TokenScopeKey).(string); isAPIToken {
				next.ServeHTTP(w, r)
				return
			}
			sessionID, _ := r.Context().Value(SessionIDKey).(string)
			userID := GetUserID(r.Context())
			if sessionID == "" || userID == "" {
				sendJSONError(w, http.StatusUnauthorized, "Session must be renewed")
				return
			}
			active, err := checker.IsSessionActive(r.Context(), userID, sessionID)
			if err != nil {
				sendJSONError(w, http.StatusInternalServerError, "Session check failed")
				return
			}
			if !active {
				sendJSONError(w, http.StatusUnauthorized, "Session is no longer active")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type APITokenValidator interface {
	ValidateAPIToken(ctx context.Context, tokenHash string) (userID, email, scope string, err error)
}

// AuthenticateJWT returns a middleware that accepts a session access token only. An API token
// (auth.TokenPrefix) is refused here: it only opens the integration routes (AuthenticateIntegration),
// so a token copied into a home-automation setup cannot read the account, manage sessions or mint tokens.
func AuthenticateJWT(jwtSecret string) func(http.Handler) http.Handler {
	return authenticate(jwtSecret, nil)
}

// AuthenticateIntegration returns a middleware for the integration routes (/api/integrations/**):
// it accepts an API token, checked by validator, as well as a session access token.
func AuthenticateIntegration(jwtSecret string, validator APITokenValidator) func(http.Handler) http.Handler {
	return authenticate(jwtSecret, validator)
}

// authenticate validates the bearer token; API tokens are accepted only when validator is set.
func authenticate(jwtSecret string, tokenValidator APITokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractBearerToken(r)
			if err != nil {
				sendJSONError(w, http.StatusUnauthorized, err.Error())
				return
			}

			if auth.IsAPIToken(tokenString) {
				if tokenValidator == nil {
					sendJSONError(w, http.StatusUnauthorized, "API tokens are only accepted on integration endpoints")
					return
				}
				tokenHash := auth.HashAPIToken(tokenString)
				grant, err := ValidateTokenGrant(r.Context(), tokenValidator, tokenHash)
				if err != nil {
					sendJSONError(w, http.StatusUnauthorized, "Invalid or expired API token")
					return
				}
				ctx := context.WithValue(r.Context(), UserIDKey, grant.UserID)
				ctx = context.WithValue(ctx, UserEmailKey, grant.Email)
				ctx = context.WithValue(ctx, TokenScopeKey, grant.Scope)
				ctx = WithVehicleScope(ctx, grant.VehicleIDs)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			claims, err := auth.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				sendJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, SessionIDKey, claims.SessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTokenScope lets a request through when it comes from a session, or from an API token whose scope is one of
// scopes. A token of another scope gets a 403: it is valid, it is not allowed here. Run it after AuthenticateIntegration.
func RequireTokenScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if scope, isToken := r.Context().Value(TokenScopeKey).(string); isToken && !slices.Contains(scopes, scope) {
				sendJSONError(w, http.StatusForbidden, "This API token does not have the scope this route needs")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val
	}
	return ""
}

// GetUserEmail retrieves the authenticated user email from context.
func GetUserEmail(ctx context.Context) string {
	if val, ok := ctx.Value(UserEmailKey).(string); ok {
		return val
	}
	return ""
}

func sendJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
