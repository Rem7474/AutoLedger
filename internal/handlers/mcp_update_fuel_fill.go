package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpUpdateFuelFillInput struct {
	VehicleID       string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	FillID          string   `json:"fill_id" jsonschema:"the id of the fill-up, as list_expenses (kind fuel_fill_up) or add_fuel_fill gives it"`
	ExpectedVersion *int64   `json:"expected_version,omitempty" jsonschema:"the version the dry run showed; required to save, and the call is refused when the fill-up was saved since"`
	Date            *string  `json:"date,omitempty" jsonschema:"the day of the fill-up, YYYY-MM-DD"`
	Amount          *string  `json:"amount,omitempty" jsonschema:"what it cost in all, a decimal string in the vehicle's currency"`
	Liters          *float64 `json:"liters,omitempty" jsonschema:"the quantity of fuel, between 0.01 and 500 litres"`
	PricePerLiter   *float64 `json:"price_per_liter,omitempty" jsonschema:"the price of a litre, in the vehicle's currency"`
	OdometerKm      *float64 `json:"odometer_km,omitempty" jsonschema:"what the odometer showed, in kilometres"`
	FuelType        *string  `json:"fuel_type,omitempty" jsonschema:"SP95_E10, SP98, DIESEL, E85 or GPL"`
	IsFullTank      *bool    `json:"is_full_tank,omitempty" jsonschema:"false for a partial fill-up"`
	Notes           *string  `json:"notes,omitempty" jsonschema:"a free note, 500 characters at most; an empty string removes it"`
	DryRun          bool     `json:"dry_run,omitempty" jsonschema:"true to see the fill-up, its version and the change, without saving anything"`
}

// IntegrationFuelFillUpdated is the answer of update_fuel_fill.
type IntegrationFuelFillUpdated struct {
	Updated bool                `json:"updated" jsonschema:"false for a dry run"`
	Version int64               `json:"version" jsonschema:"the version now; the one to give back as expected_version on a dry run"`
	Before  IntegrationFuelFill `json:"before" jsonschema:"the fill-up as it was"`
	After   IntegrationFuelFill `json:"after" jsonschema:"the fill-up as it is, or would be, saved"`
}

func (h *MCPHandler) addFuelUpdateTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "update_fuel_fill",
		Title: "Change a fuel fill-up",
		Description: "Changes a fuel fill-up the owner recorded (found with list_expenses): give only what changes. The amount, the litres and the price of a litre stay consistent: changing one of them works the others out from what is kept. " +
			"Call it with dry_run true first: it shows the fill-up, its version and the change. Show it to the owner, and save only once they agree, with that version as expected_version; the call is refused when the fill-up was changed since. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, "Could not change the fill-up", func(in mcpUpdateFuelFillInput) string { return in.VehicleID }, h.read.UpdateFuelFill)
}

// UpdateFuelFill changes the given fields of a fill-up at the version the caller read, keeping the others.
func (h *IntegrationReadHandler) UpdateFuelFill(ctx context.Context, userID, vehicleID string, in mcpUpdateFuelFillInput) (*IntegrationFuelFillUpdated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a fill-up cannot be changed on it")
	}
	if !vehicle.CanRefuel() {
		return nil, apierror.New("fuel.combustion_only", "Fuel fill-ups do not apply to electric vehicles")
	}
	if in.FillID == "" {
		return nil, mcpUserError("fill_id is required: take it from list_expenses")
	}
	if in.Date == nil && in.Amount == nil && in.Liters == nil && in.PricePerLiter == nil && in.OdometerKm == nil && in.FuelType == nil && in.IsFullTank == nil && in.Notes == nil {
		return nil, mcpUserError("nothing to change: give at least one of date, amount, liters, price_per_liter, odometer_km, fuel_type, is_full_tank, notes")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to save: call with dry_run true to get it")
	}
	if in.Notes != nil && len(*in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	current, err := h.repo.GetFuelLog(ctx, vehicle.ID, in.FillID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("no such fill-up on this vehicle: nothing was changed")
	}
	if err != nil {
		return nil, err
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != current.Version {
		return nil, errRecordChanged
	}

	req := SaveFuelLogRequest{
		Date: current.Date.Format(time.RFC3339), Odometer: current.Odometer, FuelType: current.FuelType,
		IsFullTank: &current.IsFullTank, Notes: current.Notes,
	}
	if err := fuelFigures(&req, current, in); err != nil {
		return nil, err
	}
	if in.Date != nil {
		req.Date = *in.Date
	}
	if in.OdometerKm != nil {
		req.Odometer = in.OdometerKm
	}
	if in.FuelType != nil {
		req.FuelType = in.FuelType
	}
	if in.IsFullTank != nil {
		req.IsFullTank = in.IsFullTank
	}
	if in.Notes != nil {
		req.Notes = in.Notes
	}
	fill, err := buildFuelLog(vehicle.ID, &req, h.dateLocation())
	if err != nil {
		return nil, err
	}
	fill.ID = current.ID
	if fill.Odometer != nil {
		points, err := h.repo.ListManualOdometerPoints(ctx, vehicle.ID)
		if err != nil {
			return nil, err
		}
		if err := checkOdometerOrder(points, current.ID, fill.Date, *fill.Odometer); err != nil {
			return nil, err
		}
	}
	out := &IntegrationFuelFillUpdated{Version: current.Version, Before: fuelFillOf(current, vehicle)}
	if !in.DryRun {
		fill.ExpectedVersion = in.ExpectedVersion
		if err := h.repo.UpdateFuelLog(ctx, fill); errors.Is(err, database.ErrStaleWrite) {
			return nil, errRecordChanged
		} else if err != nil {
			return nil, err
		}
		out.Updated, out.Version = true, fill.Version
	}
	out.After = fuelFillOf(fill, vehicle)
	return out, nil
}

// fuelFigures fills the amount, the litres and the price of req from what the call gives and what the fill-up holds:
// two of the three decide, the third is worked out by buildFuelLog.
func fuelFigures(req *SaveFuelLogRequest, current *models.FuelLog, in mcpUpdateFuelFillInput) error {
	var given struct{ amount, liters, price bool }
	given.amount, given.liters, given.price = in.Amount != nil, in.Liters != nil, in.PricePerLiter != nil
	if given.amount {
		cents, err := expenseAmountOf(*in.Amount, "amount")
		if err != nil {
			return err
		}
		req.Amount = &cents
	}
	if given.liters {
		req.Liters = in.Liters
	}
	if given.price {
		req.PricePerLiter = in.PricePerLiter
	}
	keptAmount, keptLiters := current.Amount, current.Liters
	switch {
	case !given.amount && !given.liters && !given.price:
		req.Amount, req.Liters, req.PricePerLiter = &keptAmount, keptLiters, current.PricePerLiter
	case given.amount && !given.liters && !given.price:
		req.Liters = keptLiters
	case !given.amount && given.liters && !given.price:
		req.Amount = &keptAmount
	case !given.amount && !given.liters && given.price:
		req.Amount = &keptAmount
	}
	return nil
}
