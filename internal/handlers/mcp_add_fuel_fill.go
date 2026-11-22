package handlers

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpAddFuelFillInput struct {
	VehicleID      string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Date           string   `json:"date" jsonschema:"YYYY-MM-DD"`
	Amount         string   `json:"amount,omitempty" jsonschema:"what it cost in all, a decimal string in the vehicle's currency such as \"62.40\"; may be left out when liters and price_per_liter are given"`
	Liters         *float64 `json:"liters,omitempty" jsonschema:"the quantity of fuel, between 0.01 and 500 litres; may be left out when amount and price_per_liter are given"`
	PricePerLiter  *float64 `json:"price_per_liter,omitempty" jsonschema:"the price of a litre, in the vehicle's currency; may be left out when amount and liters are given"`
	OdometerKm     *float64 `json:"odometer_km,omitempty" jsonschema:"what the odometer showed, in kilometres; left out, the app estimates it from the readings"`
	FuelType       string   `json:"fuel_type,omitempty" jsonschema:"SP95_E10, SP98, DIESEL, E85 or GPL"`
	IsFullTank     *bool    `json:"is_full_tank,omitempty" jsonschema:"false for a partial fill-up; true when left out. A consumption is measured between two full tanks"`
	Notes          string   `json:"notes,omitempty" jsonschema:"a free note on the fill-up, 500 characters at most"`
	DryRun         bool     `json:"dry_run,omitempty" jsonschema:"true to check the fill-up and see it, with the amount, litres and price worked out, without recording anything"`
	IdempotencyKey string   `json:"idempotency_key,omitempty" jsonschema:"a value you make up, 100 characters at most: a retry with the same key and arguments returns the first answer and records nothing more; without it a retry may record twice. Not used by a dry run"`
}

// IntegrationFuelFill is a fill-up the owner recorded.
type IntegrationFuelFill struct {
	ID            string               `json:"id" jsonschema:"id; empty on a dry run"`
	Date          string               `json:"date" jsonschema:"the day of the fill-up, YYYY-MM-DD"`
	Amount        IntegrationAmount    `json:"amount" jsonschema:"total cost"`
	Liters        *float64             `json:"liters" jsonschema:"litres; null when unknown"`
	PricePerLiter *float64             `json:"price_per_liter" jsonschema:"price of a litre; null when unknown"`
	Odometer      *IntegrationDistance `json:"odometer" jsonschema:"null when not given"`
	FuelType      *string              `json:"fuel_type" jsonschema:"null when not given"`
	IsFullTank    bool                 `json:"is_full_tank" jsonschema:"full tank or not"`
	Notes         *string              `json:"notes" jsonschema:"user text; null when none"`
}

// IntegrationFuelFillRecorded is the answer of add_fuel_fill.
type IntegrationFuelFillRecorded struct {
	Recorded bool                `json:"recorded" jsonschema:"false for a dry run"`
	Fill     IntegrationFuelFill `json:"fill" jsonschema:"the fill-up as recorded, or as it would be"`
}

func (h *MCPHandler) addFuelWriteTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "add_fuel_fill",
		Title: "Record a fuel fill-up",
		Description: "Records a fill-up of a combustion or hybrid vehicle, as the fill-up form of the app does. Give the day and two of the amount, the litres and the price per litre: the third is worked out. The odometer is optional; when given it is checked against the other readings and a mileage that contradicts them is refused. " +
			"It does not apply to electric vehicles (record a charge instead) and never changes or deletes a fill-up. " +
			"This writes to the owner's records: call it with dry_run true first, show the owner the fill-up as worked out, and record only once they agree. Pass an idempotency_key so that a retry after a lost answer records once. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not record the fill-up", func(in mcpAddFuelFillInput) string { return in.VehicleID }, h.read.AddFuelFill,
		withIdempotencyKey(func(in mcpAddFuelFillInput) string { return in.IdempotencyKey }, func(in mcpAddFuelFillInput) bool { return in.DryRun }))
}

// AddFuelFill records a fill-up, validated and checked against the odometer readings as the fuel routes do.
// With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) AddFuelFill(ctx context.Context, userID, vehicleID string, in mcpAddFuelFillInput) (*IntegrationFuelFillRecorded, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a fill-up cannot be recorded on it")
	}
	if !vehicle.CanRefuel() {
		return nil, apierror.New("fuel.combustion_only", "Fuel fill-ups do not apply to electric vehicles")
	}
	if len(in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	req := SaveFuelLogRequest{
		Date: in.Date, Odometer: in.OdometerKm, Liters: in.Liters, PricePerLiter: in.PricePerLiter,
		IsFullTank: in.IsFullTank,
	}
	if strings.TrimSpace(in.Amount) != "" {
		amount, err := expenseAmountOf(in.Amount, "amount")
		if err != nil {
			return nil, err
		}
		req.Amount = &amount
	}
	if in.FuelType != "" {
		req.FuelType = &in.FuelType
	}
	if in.Notes != "" {
		req.Notes = &in.Notes
	}
	fill, err := buildFuelLog(vehicle.ID, &req, h.dateLocation())
	if err != nil {
		return nil, err
	}
	if fill.Odometer != nil {
		points, err := h.repo.ListManualOdometerPoints(ctx, vehicle.ID)
		if err != nil {
			return nil, err
		}
		if err := checkOdometerOrder(points, "", fill.Date, *fill.Odometer); err != nil {
			return nil, err
		}
	}
	if !in.DryRun {
		if err := h.repo.CreateFuelLog(ctx, fill); err != nil {
			return nil, err
		}
	}
	return &IntegrationFuelFillRecorded{Recorded: !in.DryRun, Fill: fuelFillOf(fill, vehicle)}, nil
}

func fuelFillOf(f *models.FuelLog, vehicle *models.Vehicle) IntegrationFuelFill {
	out := IntegrationFuelFill{
		ID: f.ID, Date: f.Date.Format("2006-01-02"), Amount: amountOf(f.Amount, vehicle.Currency),
		Liters: f.Liters, PricePerLiter: f.PricePerLiter, FuelType: f.FuelType, IsFullTank: f.IsFullTank, Notes: f.Notes,
	}
	if f.Odometer != nil {
		d := kilometres(*f.Odometer)
		out.Odometer = &d
	}
	return out
}
