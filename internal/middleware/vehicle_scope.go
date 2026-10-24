package middleware

import (
	"context"
	"slices"

	"github.com/teslacost/teslacost/internal/models"
)

// VehicleScopeKey holds the ids of the only vehicles the API token of the request can read; absent when the token
// is not limited, and for a session.
const VehicleScopeKey contextKey = "vehicleScope"

// APITokenGrantValidator is implemented by validators that also say which vehicles a token is limited to.
type APITokenGrantValidator interface {
	ValidateAPITokenGrant(ctx context.Context, tokenHash string) (models.TokenGrant, error)
}

// WithVehicleScope limits the request to the given vehicles; a nil or empty list leaves it unlimited.
func WithVehicleScope(ctx context.Context, vehicleIDs []string) context.Context {
	if len(vehicleIDs) == 0 {
		return ctx
	}
	return context.WithValue(ctx, VehicleScopeKey, vehicleIDs)
}

// VehicleScope returns the vehicles the request is limited to and whether it is limited at all.
func VehicleScope(ctx context.Context) (ids []string, limited bool) {
	ids, limited = ctx.Value(VehicleScopeKey).([]string)
	return ids, limited
}

// VehicleAllowed says whether the request may read the vehicle: always when it is not limited.
func VehicleAllowed(ctx context.Context, vehicleID string) bool {
	ids, limited := VehicleScope(ctx)
	return !limited || slices.Contains(ids, vehicleID)
}

// ValidateTokenGrant checks an API token with the richest method the validator offers.
func ValidateTokenGrant(ctx context.Context, v APITokenValidator, tokenHash string) (models.TokenGrant, error) {
	if gv, ok := v.(APITokenGrantValidator); ok {
		return gv.ValidateAPITokenGrant(ctx, tokenHash)
	}
	userID, email, scope, err := v.ValidateAPIToken(ctx, tokenHash)
	return models.TokenGrant{UserID: userID, Email: email, Scope: scope}, err
}
