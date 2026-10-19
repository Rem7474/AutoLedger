package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/oauth"
)

// maxOAuthClients bounds the registrations an instance keeps: registering needs no account, so it is capped.
const maxOAuthClients = 1000

const (
	oauthGrantAuthorizationCode = "authorization_code"
	oauthGrantRefreshToken      = "refresh_token"
	maxOAuthStateLength         = 512
)

// OAuthHandler is the authorization server an assistant uses to obtain a read token without the user copying one:
// the client registers itself, sends the user to the consent page, and exchanges the code the approval produces.
type OAuthHandler struct {
	repo   *database.Repository
	issuer string
}

// NewOAuthHandler serves the authorization server of the instance reachable at baseURL.
func NewOAuthHandler(repo *database.Repository, baseURL string) *OAuthHandler {
	return &OAuthHandler{repo: repo, issuer: strings.TrimRight(baseURL, "/")}
}

// Resource is the address of the MCP endpoint, the resource a token is requested for.
func (h *OAuthHandler) Resource() string { return h.issuer + "/mcp" }

// oauthError is the error body of the OAuth endpoints (RFC 6749 §5.2).
type oauthError struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

func writeOAuthJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, status, body)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	writeOAuthJSON(w, status, oauthError{Error: code, Description: description})
}

// Metadata describes the authorization server (RFC 8414), so a client finds the endpoints from the instance address.
func (h *OAuthHandler) Metadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         h.issuer,
		"authorization_endpoint":                         h.issuer + "/oauth/authorize",
		"token_endpoint":                                 h.issuer + "/oauth/token",
		"revocation_endpoint":                            h.issuer + "/oauth/revoke",
		"registration_endpoint":                          h.issuer + "/oauth/register",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{oauthGrantAuthorizationCode, oauthGrantRefreshToken},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"code_challenge_methods_supported":               []string{"S256"},
		"scopes_supported":                               oauth.Scopes,
		"authorization_response_iss_parameter_supported": true,
	})
}

type registerRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// cleanClientName removes the control characters of a name an unknown application chose, and bounds its length.
func cleanClientName(raw string) string {
	name := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, raw))
	if runes := []rune(name); len(runes) > oauth.MaxClientNameLength {
		name = string(runes[:oauth.MaxClientNameLength])
	}
	return name
}

func subsetOf(values, allowed []string) bool {
	for _, v := range values {
		found := false
		for _, a := range allowed {
			if v == a {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Register creates a public client (RFC 7591). The application states its name and where the approval may send the
// user back; both are shown to the user on the consent page.
func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "The body is not valid JSON")
		return
	}
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > oauth.MaxRedirectURIs {
		writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "Give between 1 and 10 redirect URIs")
		return
	}
	for _, uri := range req.RedirectURIs {
		if !oauth.ValidRedirectURI(uri) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "A redirect URI must be https, http on a loopback address, or an application scheme, without fragment or wildcard")
			return
		}
	}
	if req.TokenEndpointAuthMethod != "" && req.TokenEndpointAuthMethod != "none" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "Only public clients are supported: token_endpoint_auth_method must be none")
		return
	}
	if !subsetOf(req.GrantTypes, []string{oauthGrantAuthorizationCode, oauthGrantRefreshToken}) || !subsetOf(req.ResponseTypes, []string{"code"}) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "Supported grant types are authorization_code and refresh_token, the response type is code")
		return
	}
	name := cleanClientName(req.ClientName)
	if name == "" {
		name = "Unnamed application"
	}
	id, err := oauth.NewClientID()
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	client := &models.OAuthClient{ID: id, Name: name, RedirectURIs: req.RedirectURIs}
	switch err := h.repo.CreateOAuthClient(r.Context(), client, maxOAuthClients); {
	case errors.Is(err, database.ErrTooManyOAuthClients):
		writeOAuthError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "This instance accepts no more registered applications")
		return
	case err != nil:
		slog.Error("oauth client registration failed", "component", "oauth", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	writeOAuthJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ID,
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{oauthGrantAuthorizationCode, oauthGrantRefreshToken},
		"response_types":             []string{"code"},
		"client_id_issued_at":        time.Now().Unix(),
	})
}

// authorizeRequest carries the parameters of an authorization request (RFC 6749 §4.1.1, PKCE, RFC 8707).
type authorizeRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	ResponseType        string `json:"response_type"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Resource            string `json:"resource"`
	// Decision is "approve" or "deny"; it is read on the POST only.
	Decision string `json:"decision"`
}

func authorizeRequestFromQuery(q url.Values) authorizeRequest {
	return authorizeRequest{
		ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"), ResponseType: q.Get("response_type"),
		CodeChallenge: q.Get("code_challenge"), CodeChallengeMethod: q.Get("code_challenge_method"),
		Scope: q.Get("scope"), State: q.Get("state"), Resource: q.Get("resource"),
	}
}

// check validates an authorization request against the registered client and returns the client and the scope.
// Nothing is redirected on failure: when the request itself is wrong, the user is told on the consent page.
func (h *OAuthHandler) check(r *http.Request, p authorizeRequest) (*models.OAuthClient, string, *apierror.Error) {
	if p.ResponseType != "code" || p.ClientID == "" || p.RedirectURI == "" {
		return nil, "", apierror.New("oauth.invalid_request", "The authorization request is incomplete")
	}
	client, err := h.repo.GetOAuthClient(r.Context(), p.ClientID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, "", apierror.New("oauth.unknown_client", "This application is not registered")
	}
	if err != nil {
		return nil, "", apierror.New("internal", "Failed to load the application")
	}
	if !oauth.MatchRedirectURI(client.RedirectURIs, p.RedirectURI) {
		return nil, "", apierror.New("oauth.invalid_redirect_uri", "The return address does not match the one the application registered")
	}
	if p.CodeChallengeMethod != "S256" || !oauth.ValidChallenge(p.CodeChallenge) {
		return nil, "", apierror.New("oauth.invalid_request", "The authorization request must use PKCE with the S256 method")
	}
	scope, ok := oauth.NormalizeScope(p.Scope)
	if !ok {
		return nil, "", apierror.New("oauth.invalid_scope", "The application asks for access that does not exist")
	}
	if p.Resource != "" && p.Resource != h.Resource() {
		return nil, "", apierror.New("oauth.invalid_resource", "The application asks for a token for another server")
	}
	if len(p.State) > maxOAuthStateLength {
		return nil, "", apierror.New("oauth.invalid_request", "The authorization request is invalid")
	}
	return client, scope, nil
}

// PreviewAuthorization says who is asking and for what, for the consent page. It writes nothing.
func (h *OAuthHandler) PreviewAuthorization(w http.ResponseWriter, r *http.Request) {
	p := authorizeRequestFromQuery(r.URL.Query())
	client, scope, apiErr := h.check(r, p)
	if apiErr != nil {
		writeAPIError(w, http.StatusBadRequest, apiErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"client_name":   client.Name,
		"redirect_host": oauth.RedirectHost(p.RedirectURI),
		"scope":         scope,
	})
}

// DecideAuthorization records the user's decision and answers with the address to send the browser to: the client's
// return address with an authorization code, or with access_denied. The request is validated again, whatever the
// preview said.
func (h *OAuthHandler) DecideAuthorization(w http.ResponseWriter, r *http.Request) {
	var p authorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	client, scope, apiErr := h.check(r, p)
	if apiErr != nil {
		writeAPIError(w, http.StatusBadRequest, apiErr)
		return
	}
	target, err := url.Parse(p.RedirectURI)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("oauth.invalid_redirect_uri", "The return address does not match the one the application registered"))
		return
	}
	query := target.Query()
	switch p.Decision {
	case "deny":
		query.Set("error", "access_denied")
	case "approve":
		code, hash, err := oauth.NewCode()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, apierror.New("internal", "Failed to authorize"))
			return
		}
		stored := models.OAuthCode{ClientID: client.ID, UserID: middleware.GetUserID(r.Context()), RedirectURI: p.RedirectURI, CodeChallenge: p.CodeChallenge, Scope: scope}
		if err := h.repo.CreateOAuthCode(r.Context(), hash, stored, time.Now().Add(oauth.CodeTTL)); err != nil {
			writeRepoError(w, r, err, "Failed to authorize")
			return
		}
		query.Set("code", code)
	default:
		writeAPIError(w, http.StatusBadRequest, apierror.New("oauth.invalid_request", "The authorization request is invalid"))
		return
	}
	if p.State != "" {
		query.Set("state", p.State)
	}
	query.Set("iss", h.issuer)
	target.RawQuery = query.Encode()
	writeJSON(w, http.StatusOK, map[string]string{"redirect_url": target.String()})
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// newGrantTokens generates the secrets of a grant and the response that hands them to the client.
func newGrantTokens(userID, clientID, name, scope string, now time.Time) (database.OAuthGrant, tokenResponse, error) {
	access, prefix, accessHash, err := auth.GenerateAPIToken()
	if err != nil {
		return database.OAuthGrant{}, tokenResponse{}, err
	}
	refresh, refreshHash, err := oauth.NewRefreshToken()
	if err != nil {
		return database.OAuthGrant{}, tokenResponse{}, err
	}
	grant := database.OAuthGrant{
		UserID: userID, ClientID: clientID, Name: name, Scope: scope,
		AccessHash: accessHash, AccessPrefix: prefix, AccessExpiresAt: now.Add(oauth.AccessTokenTTL),
		RefreshHash: refreshHash, RefreshExpiresAt: now.Add(oauth.RefreshTokenTTL),
	}
	return grant, tokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: int(oauth.AccessTokenTTL.Seconds()), RefreshToken: refresh, Scope: scope}, nil
}

// Token exchanges an authorization code or a refresh token for an access token (RFC 6749 §4.1.3, §6).
func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "The body must be form encoded")
		return
	}
	switch r.PostForm.Get("grant_type") {
	case oauthGrantAuthorizationCode:
		h.exchangeCode(w, r)
	case oauthGrantRefreshToken:
		h.refresh(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (h *OAuthHandler) exchangeCode(w http.ResponseWriter, r *http.Request) {
	code, redirectURI, clientID, verifier := r.PostForm.Get("code"), r.PostForm.Get("redirect_uri"), r.PostForm.Get("client_id"), r.PostForm.Get("code_verifier")
	if code == "" || redirectURI == "" || clientID == "" || verifier == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code, redirect_uri, client_id and code_verifier are required")
		return
	}
	// The code is consumed before anything is checked: a wrong verifier or address spends it.
	stored, err := h.repo.ConsumeOAuthCode(r.Context(), oauth.Hash(code))
	if errors.Is(err, database.ErrNotFound) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "The authorization code is invalid, expired or already used")
		return
	}
	if err != nil {
		slog.Error("oauth code exchange failed", "component", "oauth", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	if stored.ClientID != clientID || stored.RedirectURI != redirectURI || !oauth.VerifyPKCE(verifier, stored.CodeChallenge) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "The authorization code does not match this request")
		return
	}
	client, err := h.repo.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "This application is not registered")
		return
	}
	grant, resp, err := newGrantTokens(stored.UserID, client.ID, client.Name, stored.Scope, time.Now())
	if err == nil {
		err = h.repo.IssueOAuthGrant(r.Context(), grant)
	}
	if err != nil {
		slog.Error("oauth grant could not be issued", "component", "oauth", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	writeOAuthJSON(w, http.StatusOK, resp)
}

func (h *OAuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	presented, clientID := r.PostForm.Get("refresh_token"), r.PostForm.Get("client_id")
	if presented == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "refresh_token and client_id are required")
		return
	}
	next, resp, err := newGrantTokens("", clientID, "", "", time.Now())
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	scope, err := h.repo.RotateOAuthGrant(r.Context(), clientID, oauth.Hash(presented), next)
	switch {
	case errors.Is(err, database.ErrNotFound):
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "The refresh token is invalid or expired")
		return
	case errors.Is(err, database.ErrOAuthRefreshReused):
		slog.Warn("oauth refresh token reused, grant revoked", "component", "oauth", "client_id", clientID)
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "The refresh token was already used; authorize the application again")
		return
	case err != nil:
		slog.Error("oauth refresh failed", "component", "oauth", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	resp.Scope = scope
	writeOAuthJSON(w, http.StatusOK, resp)
}

// Revoke ends a grant from the access or the refresh token it holds (RFC 7009). An unknown token is not an error.
func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "The body must be form encoded")
		return
	}
	token, clientID := r.PostForm.Get("token"), r.PostForm.Get("client_id")
	if token == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token and client_id are required")
		return
	}
	if err := h.repo.RevokeOAuthGrant(r.Context(), clientID, oauth.Hash(token)); err != nil {
		slog.Error("oauth revocation failed", "component", "oauth", "error", err)
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}
	writeOAuthJSON(w, http.StatusOK, map[string]any{})
}
