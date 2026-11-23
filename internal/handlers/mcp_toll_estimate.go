package handlers

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/services"
)

type mcpEstimateTollsInput struct {
	VehicleID string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	DriveIDs  []string `json:"drive_ids" jsonschema:"the trips to estimate the tolls of, by the drive_id list_vehicle_trips gives, 100 at most"`
}

// IntegrationTollEstimate is what became of one trip.
type IntegrationTollEstimate struct {
	DriveID string             `json:"drive_id" jsonschema:"the trip"`
	Status  string             `json:"status" jsonschema:"created or updated: an estimated toll was recorded; skipped_manual: the owner entered the tolls, kept; skipped_trip_group: the trip's tolls belong to its trip group, kept; skipped_no_price: no toll fare is known for the road taken; skipped_no_gps: the trip has no GPS trace; skipped_currency: the vehicle's currency has no toll fares; failed"`
	Amount  *IntegrationAmount `json:"amount" jsonschema:"the estimated toll recorded; null when none"`
}

// IntegrationTollEstimates is the answer of estimate_trip_tolls.
type IntegrationTollEstimates struct {
	Created int                       `json:"created" jsonschema:"trips that got an estimated toll"`
	Updated int                       `json:"updated" jsonschema:"trips whose earlier estimate was replaced"`
	Skipped int                       `json:"skipped" jsonschema:"trips left as they were"`
	Failed  int                       `json:"failed" jsonschema:"trips the estimate could not be worked out for"`
	Trips   []IntegrationTollEstimate `json:"trips" jsonschema:"what became of each trip, in the order given"`
}

func (h *MCPHandler) addTollEstimateTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "estimate_trip_tolls",
		Title: "Estimate the tolls of trips from their trace",
		Description: "Detects the tolls a trip went through from its GPS trace and records the estimated total as the trip's toll, as the trips page does. It needs the vehicle to read its trace from TeslaMate and toll detection turned on for it, and uses the known fare table. " +
			"It never overwrites tolls the owner entered or tolls shared by a trip group: those trips are reported as skipped. A trip estimated before is estimated again. Tell the owner which trips are concerned and call it only once they agree. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, "Could not estimate the tolls", func(in mcpEstimateTollsInput) string { return in.VehicleID }, h.read.EstimateTripTolls)
}

// EstimateTripTolls applies the estimated toll to the trips named.
func (h *IntegrationReadHandler) EstimateTripTolls(ctx context.Context, userID, vehicleID string, in mcpEstimateTollsInput) (*IntegrationTollEstimates, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: tolls cannot be estimated on it")
	}
	if h.tolls == nil {
		return nil, errors.New("toll detection unavailable")
	}
	if !vehicle.TollDetectionActive {
		return nil, mcpUserError("toll detection is turned off for this vehicle: the owner turns it on in the vehicle's settings")
	}
	if len(in.DriveIDs) == 0 || len(in.DriveIDs) > services.MaxBulkTollDrives {
		return nil, mcpUserError("drive_ids holds 1 to 100 trips")
	}
	result := h.tolls.ApplyTollEstimatesBulk(ctx, vehicle, in.DriveIDs)
	out := &IntegrationTollEstimates{
		Created: result.Created, Updated: result.Updated, Failed: result.Failed,
		Skipped: result.SkippedManual + result.SkippedTripGroup + result.SkippedNoPrice + result.SkippedNoGPS + result.SkippedCurrency,
		Trips:   make([]IntegrationTollEstimate, 0, len(result.Results)),
	}
	for _, r := range result.Results {
		item := IntegrationTollEstimate{DriveID: r.DriveID, Status: string(r.Status)}
		if r.Amount != nil {
			a := amountOf(*r.Amount, vehicle.Currency)
			item.Amount = &a
		}
		out.Trips = append(out.Trips, item)
	}
	return out, nil
}
