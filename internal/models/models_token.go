package models

import "time"

// APIToken represents a Personal Access Token used for API / Home Assistant access.
type APIToken struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	TokenPrefix string `json:"token_prefix"`
	Scope       string `json:"scope"` // "ingest" or "read", fixed at creation
	// OAuth is true for a grant an assistant obtained through the authorization server; ExpiresAt is then when the
	// grant lapses unless the application renews it.
	OAuth bool `json:"oauth"`
	// VehicleIDs are the only vehicles the token can read; null is every vehicle of the account.
	VehicleIDs []string   `json:"vehicle_ids"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateAPITokenRequest is sent when generating a new token.
type CreateAPITokenRequest struct {
	Name      string     `json:"name"`
	Scope     string     `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APITokenCreatedResponse returns the generated clear-text token once.
type APITokenCreatedResponse struct {
	Token string   `json:"token"`
	Info  APIToken `json:"info"`
}

// TokenGrant is what a valid API token lets its holder do.
type TokenGrant struct {
	UserID, Email, Scope string
	// TokenID identifies the token, to attach what it does to it.
	TokenID string
	// VehicleIDs are the only vehicles the token can read; nil is every vehicle the account can see.
	VehicleIDs []string
}

// APITokenCall is one call an assistant made with a token. It holds no argument and no answer.
type APITokenCall struct {
	At         time.Time `json:"at"`
	Tool       string    `json:"tool"`
	VehicleID  *string   `json:"vehicle_id"`
	Outcome    string    `json:"outcome"` // "ok" or "error"
	DurationMS int       `json:"duration_ms"`
}
