package handlers

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpAddChargeInput struct {
	VehicleID         string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Date              string   `json:"date" jsonschema:"the day of the charge, YYYY-MM-DD"`
	KwhAdded          float64  `json:"kwh_added" jsonschema:"the energy added to the battery, in kWh, more than zero"`
	Cost              string   `json:"cost" jsonschema:"what the charge cost in all, a decimal string in the vehicle's currency such as \"7.80\"; \"0\" for a free charge"`
	OdometerKm        *float64 `json:"odometer_km,omitempty" jsonschema:"what the odometer showed, in kilometres"`
	StartBatteryLevel *int     `json:"start_battery_level,omitempty" jsonschema:"the battery percentage before the charge, 0 to 100"`
	EndBatteryLevel   *int     `json:"end_battery_level,omitempty" jsonschema:"the battery percentage after the charge, 0 to 100, higher than the start"`
	Notes             string   `json:"notes,omitempty" jsonschema:"a free note on the charge, 500 characters at most"`
	DryRun            bool     `json:"dry_run,omitempty" jsonschema:"true to check the charge and see it without recording anything"`
	IdempotencyKey    string   `json:"idempotency_key,omitempty" jsonschema:"a value you make up, 100 characters at most: a retry with the same key and arguments returns the first answer and records nothing more; without it a retry may record twice. Not used by a dry run"`
}

// IntegrationChargeRecorded is the answer of add_charge.
type IntegrationChargeRecorded struct {
	Recorded bool                 `json:"recorded" jsonschema:"false for a dry run"`
	ID       string               `json:"id" jsonschema:"id of the charge; empty on a dry run"`
	Date     string               `json:"date" jsonschema:"YYYY-MM-DD"`
	Energy   float64              `json:"kwh_added" jsonschema:"kWh added"`
	Cost     IntegrationAmount    `json:"cost" jsonschema:"total cost"`
	Odometer *IntegrationDistance `json:"odometer" jsonschema:"null when not given"`
	Notes    *string              `json:"notes" jsonschema:"user text; null when none"`
}

func (h *MCPHandler) addChargeWriteTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "add_charge",
		Title: "Record a charging session",
		Description: "Records a charge the owner paid for and that no synchronised source holds (a public terminal, a charge at someone else's home), as the manual charge form of the app does: the energy added and what it cost, which is required (\"0\" for a free charge). " +
			"It applies to vehicles that charge, never changes or deletes a charge, and does not touch the charges synchronised from TeslaMate or Home Assistant. " +
			"This writes to the owner's records: call it with dry_run true first, show the owner the charge, and record only once they agree. Pass an idempotency_key so that a retry after a lost answer records once. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not record the charge", func(in mcpAddChargeInput) string { return in.VehicleID }, h.read.AddCharge,
		withIdempotencyKey(func(in mcpAddChargeInput) string { return in.IdempotencyKey }, func(in mcpAddChargeInput) bool { return in.DryRun }))
}

// AddCharge records a manual charge, validated as the charge form of the app does. With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) AddCharge(ctx context.Context, userID, vehicleID string, in mcpAddChargeInput) (*IntegrationChargeRecorded, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a charge cannot be recorded on it")
	}
	if !vehicle.CanCharge() {
		return nil, mcpUserError("this vehicle does not charge: a charge cannot be recorded on it")
	}
	if len(in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	if strings.TrimSpace(in.Cost) == "" {
		return nil, mcpUserError("cost is required: give \"0\" for a free charge")
	}
	cost, err := expenseAmountOf(in.Cost, "cost")
	if err != nil {
		return nil, err
	}
	if in.KwhAdded <= 0 {
		return nil, apierror.New("charge.energy_positive", "The energy added must be positive")
	}
	req := SaveChargeRequest{
		Date: in.Date, KwhAdded: in.KwhAdded, Cost: &cost, Odometer: in.OdometerKm,
		StartBatteryLevel: in.StartBatteryLevel, EndBatteryLevel: in.EndBatteryLevel,
	}
	if note := strings.TrimSpace(in.Notes); note != "" {
		req.Notes = &note
	}
	charge, err := buildCharge(vehicle.ID, vehicle.Currency, &req)
	if err != nil {
		return nil, err
	}
	if !in.DryRun {
		if err := h.repo.CreateManualCharge(ctx, charge); err != nil {
			return nil, err
		}
	}
	return chargeRecordedOf(charge, vehicle, !in.DryRun), nil
}

func chargeRecordedOf(c *models.ChargeLog, vehicle *models.Vehicle, recorded bool) *IntegrationChargeRecorded {
	out := &IntegrationChargeRecorded{
		Recorded: recorded, ID: c.ID, Date: c.Date.Format("2006-01-02"), Energy: c.KwhAdded,
		Cost: amountOf(*c.Cost, vehicle.Currency), Notes: c.Notes,
	}
	if c.Odometer != nil {
		d := kilometres(*c.Odometer)
		out.Odometer = &d
	}
	return out
}
