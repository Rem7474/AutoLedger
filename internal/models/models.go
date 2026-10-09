package models

import (
	"time"
)

// AuthMode represents authentication type for TeslaMate API.
type AuthMode string

const (
	AuthModeNone   AuthMode = "NONE"
	AuthModeBearer AuthMode = "BEARER"
	AuthModeBasic  AuthMode = "BASIC"
)

// User represents a registered user.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash *string   `json:"-"` // nullable: OIDC accounts have no local password
	OIDCSubject  *string   `json:"-"`
	OIDCProvider *string   `json:"-"`
	DisplayName  *string   `json:"display_name,omitempty"` // from IdP "name" claim
	Language     string    `json:"language"`               // "en" or "fr": used for messages built outside a request (reminder webhooks, sync alerts)
	DistanceUnit string    `json:"distance_unit"`          // "km" or "mi": distances are always stored in km, this only drives display/input conversion
	VolumeUnit   string    `json:"volume_unit"`            // "l", "gal_us" or "gal_uk": volumes are always stored in litres, this only drives display/input conversion
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken represents a long-lived refresh token session with rotation family tracking.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	FamilyID  string    `json:"family_id"`
	IsRevoked bool      `json:"is_revoked"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	CreatedIP *string   `json:"created_ip,omitempty"`
	UserAgent *string   `json:"user_agent,omitempty"`
}

// BatterySnapshot is the battery health computed by TeslaMate on a given day.
type BatterySnapshot struct {
	Date               string   `json:"date"` // YYYY-MM-DD
	MaxCapacityKwh     *float64 `json:"max_capacity_kwh,omitempty"`
	CurrentCapacityKwh *float64 `json:"current_capacity_kwh,omitempty"`
	HealthPercent      *float64 `json:"health_percent,omitempty"`
}

// Session is a signed-in device: a family of rotating refresh tokens.
type Session struct {
	ID         string    `json:"id"` // The refresh token family
	StartedAt  time.Time `json:"started_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	IP         *string   `json:"ip,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	Current    bool      `json:"current"`
}
