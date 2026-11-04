package handlers

import (
	"context"
	"net/http"

	"github.com/teslacost/teslacost/internal/middleware"
)

// IntegrationWhoAmI tells a client what its credential is and how the account reads its figures, so an integration
// can validate a token when it connects and say precisely what is missing ("read-only token", "ingest-only token").
type IntegrationWhoAmI struct {
	Scope           string `json:"scope" jsonschema:"what the token may do: ingest, read, homeassistant or write; session for a signed-in user"`
	Name            string `json:"name" jsonschema:"the account's display name, or its email address when it has none"`
	VehiclesLimited bool   `json:"vehicles_limited" jsonschema:"true when the token only sees some of the account's vehicles"`
	ServerVersion   string `json:"server_version" jsonschema:"version of the AutoLedger server, dev for a build from the sources"`
	PendingCharges  int    `json:"pending_charges" jsonschema:"charging sessions pushed to the account that no vehicle claimed yet (all vehicles of the account together)"`
	IntegrationAccount
}

// WithVersion gives the whoami view the version of the server.
func (h *IntegrationReadHandler) WithVersion(version string) *IntegrationReadHandler {
	h.version = version
	return h
}

// WhoAmI describes the credential of the request and the account behind it. scope is empty for a session.
func (h *IntegrationReadHandler) WhoAmI(ctx context.Context, userID, scope string) (*IntegrationWhoAmI, error) {
	account, err := h.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	pending, err := h.repo.ListPendingCharges(ctx, userID)
	if err != nil {
		return nil, err
	}
	_, limited := middleware.VehicleScope(ctx)
	if scope == "" {
		scope = "session"
	}
	name := user.Email
	if user.DisplayName != nil && *user.DisplayName != "" {
		name = *user.DisplayName
	}
	return &IntegrationWhoAmI{
		Scope: scope, Name: name, VehiclesLimited: limited, ServerVersion: h.version, PendingCharges: len(pending),
		IntegrationAccount: *account,
	}, nil
}

// GetWhoAmI serves WhoAmI.
func (h *IntegrationReadHandler) GetWhoAmI(w http.ResponseWriter, r *http.Request) {
	scope, _ := r.Context().Value(middleware.TokenScopeKey).(string)
	out, err := h.WhoAmI(r.Context(), middleware.GetUserID(r.Context()), scope)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to read the account")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
