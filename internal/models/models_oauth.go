package models

import "time"

// OAuthClient is an application (an MCP client) that registered itself to ask for access. It is public: it holds no
// secret, PKCE binds each authorization to the one that asked.
type OAuthClient struct {
	ID           string    `json:"client_id"`
	Name         string    `json:"client_name"`
	RedirectURIs []string  `json:"redirect_uris"`
	CreatedAt    time.Time `json:"-"`
}

// OAuthCode is an authorization code the user approved, waiting to be exchanged for a grant.
type OAuthCode struct {
	ClientID      string
	UserID        string
	RedirectURI   string
	CodeChallenge string
	Scope         string
	// VehicleIDs limits the grant to these vehicles; nil is every vehicle the account can see.
	VehicleIDs []string
}
