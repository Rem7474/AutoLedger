package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpUpdateChargeInput struct {
	VehicleID         string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	ChargeID          string   `json:"charge_id" jsonschema:"the id of the charge, as list_expenses (kind charge) or add_charge gives it"`
	ExpectedVersion   *int64   `json:"expected_version,omitempty" jsonschema:"the version the dry run showed; required to save, and the call is refused when the charge was saved since"`
	Cost              *string  `json:"cost,omitempty" jsonschema:"what the charge cost in all, a decimal string in the vehicle's currency; \"0\" for a free charge. The only figure a synchronised charge lets you change, with the note"`
	CostReviewed      *bool    `json:"cost_reviewed,omitempty" jsonschema:"true to confirm that the 0 cost of a synchronised charge is right (it stops being listed as to complete), false to reopen it"`
	Date              *string  `json:"date,omitempty" jsonschema:"manual charge only: the day of the charge, YYYY-MM-DD"`
	KwhAdded          *float64 `json:"kwh_added,omitempty" jsonschema:"manual charge only: the energy added, in kWh, more than zero"`
	OdometerKm        *float64 `json:"odometer_km,omitempty" jsonschema:"manual charge only: what the odometer showed, in kilometres"`
	StartBatteryLevel *int     `json:"start_battery_level,omitempty" jsonschema:"manual charge only: the battery percentage before, 0 to 100"`
	EndBatteryLevel   *int     `json:"end_battery_level,omitempty" jsonschema:"manual charge only: the battery percentage after, 0 to 100, higher than the start"`
	Notes             *string  `json:"notes,omitempty" jsonschema:"a free note, 500 characters at most; an empty string removes it"`
	DryRun            bool     `json:"dry_run,omitempty" jsonschema:"true to see the charge, its version and the change, without saving anything"`
}

// IntegrationChargeEntry is a charge as update_charge shows it.
type IntegrationChargeEntry struct {
	ID           string               `json:"id" jsonschema:"id of the charge"`
	Date         string               `json:"date" jsonschema:"YYYY-MM-DD"`
	Energy       float64              `json:"kwh_added" jsonschema:"kWh added"`
	Cost         *IntegrationAmount   `json:"cost" jsonschema:"total cost; null when it is still to be completed"`
	CostToReview bool                 `json:"cost_to_review" jsonschema:"true while the cost is unknown or a 0 nobody confirmed"`
	Synchronised bool                 `json:"synchronised" jsonschema:"true for a charge that comes from TeslaMate or Home Assistant"`
	Odometer     *IntegrationDistance `json:"odometer" jsonschema:"null when not given"`
	Notes        *string              `json:"notes" jsonschema:"user text; null when none"`
}

// IntegrationChargeUpdated is the answer of update_charge.
type IntegrationChargeUpdated struct {
	Updated bool                   `json:"updated" jsonschema:"false for a dry run"`
	Version int64                  `json:"version" jsonschema:"the version now; the one to give back as expected_version on a dry run"`
	Before  IntegrationChargeEntry `json:"before" jsonschema:"the charge as it was"`
	After   IntegrationChargeEntry `json:"after" jsonschema:"the charge as it is, or would be, saved"`
}

func (h *MCPHandler) addChargeUpdateTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "update_charge",
		Title: "Change a charge, or complete its cost",
		Description: "Changes a charge of a vehicle (found with list_expenses): the cost, a note, and for a manual charge also its day, energy, odometer and battery levels. A charge synchronised from TeslaMate or Home Assistant only lets you set its cost and note, or confirm with cost_reviewed that its 0 cost is right. " +
			"Call it with dry_run true first: it shows the charge, its version and the change. Show it to the owner, and save only once they agree, with that version as expected_version; the call is refused when the charge was changed since. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, "Could not change the charge", func(in mcpUpdateChargeInput) string { return in.VehicleID }, h.read.UpdateCharge)
}

// UpdateCharge changes the given fields of a charge at the version the caller read, keeping the others.
func (h *IntegrationReadHandler) UpdateCharge(ctx context.Context, userID, vehicleID string, in mcpUpdateChargeInput) (*IntegrationChargeUpdated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a charge cannot be changed on it")
	}
	if in.ChargeID == "" {
		return nil, mcpUserError("charge_id is required: take it from list_expenses")
	}
	figures := in.Date != nil || in.KwhAdded != nil || in.OdometerKm != nil || in.StartBatteryLevel != nil || in.EndBatteryLevel != nil
	if in.Cost == nil && in.CostReviewed == nil && in.Notes == nil && !figures {
		return nil, mcpUserError("nothing to change: give at least one of cost, cost_reviewed, notes, date, kwh_added, odometer_km, start_battery_level, end_battery_level")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to save: call with dry_run true to get it")
	}
	if in.Notes != nil && len(*in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	current, err := h.repo.GetCharge(ctx, vehicle.ID, in.ChargeID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("no such charge on this vehicle: nothing was changed")
	}
	if err != nil {
		return nil, err
	}
	if figures && !current.IsManual {
		return nil, mcpUserError("this charge is synchronised: only its cost and note can be changed (date, energy, odometer and battery levels come from the source)")
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != current.Version {
		return nil, errRecordChanged
	}

	next := *current
	if in.Cost != nil {
		cost, err := expenseAmountOf(*in.Cost, "cost")
		if err != nil {
			return nil, err
		}
		next.Cost = &cost
	}
	if in.Notes != nil {
		next.Notes = nil
		if note := strings.TrimSpace(*in.Notes); note != "" {
			next.Notes = &note
		}
	}
	if figures {
		req := SaveChargeRequest{
			Date: current.Date.Format(time.RFC3339), KwhAdded: current.KwhAdded, Cost: next.Cost, Currency: current.Currency, FxRate: current.FxRate,
			Odometer: current.Odometer, StartBatteryLevel: current.StartBatteryLevel, EndBatteryLevel: current.EndBatteryLevel,
			Address: current.Address, PlaceID: current.PlaceID, Notes: next.Notes, DocumentID: current.DocumentID,
		}
		if current.EndDate != nil {
			end := current.EndDate.Format(time.RFC3339)
			req.EndDate = &end
		}
		if in.Date != nil {
			req.Date, req.EndDate = *in.Date, nil // the end belonged to the old day
		}
		if in.KwhAdded != nil {
			if *in.KwhAdded <= 0 {
				return nil, mcpUserError("kwh_added must be more than zero")
			}
			req.KwhAdded = *in.KwhAdded
		}
		if in.OdometerKm != nil {
			req.Odometer = in.OdometerKm
		}
		if in.StartBatteryLevel != nil {
			req.StartBatteryLevel = in.StartBatteryLevel
		}
		if in.EndBatteryLevel != nil {
			req.EndBatteryLevel = in.EndBatteryLevel
		}
		built, err := buildCharge(vehicle.ID, vehicle.Currency, &req)
		if err != nil {
			return nil, err
		}
		next.Date, next.EndDate, next.KwhAdded, next.Odometer = built.Date, built.EndDate, built.KwhAdded, built.Odometer
		next.StartBatteryLevel, next.EndBatteryLevel = built.StartBatteryLevel, built.EndBatteryLevel
	}
	if next.Cost == nil {
		return nil, mcpUserError("this charge has no cost yet: give cost (\"0\" for a free charge)")
	}

	out := &IntegrationChargeUpdated{Version: current.Version, Before: chargeEntryOf(current, vehicle)}
	reviewed := current.CostReviewedAt != nil
	if in.CostReviewed != nil {
		reviewed = *in.CostReviewed
	}
	if !in.DryRun {
		next.ExpectedVersion = in.ExpectedVersion
		changed := in.Cost != nil || in.Notes != nil || figures
		if changed {
			if err := h.repo.UpdateCharge(ctx, &next); errors.Is(err, database.ErrStaleWrite) {
				return nil, errRecordChanged
			} else if err != nil {
				return nil, err
			}
		}
		if in.CostReviewed != nil {
			if err := h.repo.SetChargeCostReviewed(ctx, current.ID, vehicle.ID, *in.CostReviewed); err != nil {
				return nil, err
			}
		}
		fresh, err := h.repo.GetCharge(ctx, vehicle.ID, current.ID)
		if err != nil {
			return nil, err
		}
		out.Updated, out.Version = true, fresh.Version
		out.After = chargeEntryOf(fresh, vehicle)
		return out, nil
	}
	next.CostReviewedAt = nil
	if reviewed {
		now := time.Now()
		next.CostReviewedAt = &now
	}
	next.CostToReview = next.Cost == nil || (*next.Cost == 0 && !current.IsManual && !reviewed)
	out.After = chargeEntryOf(&next, vehicle)
	return out, nil
}

func chargeEntryOf(c *models.ChargeLog, vehicle *models.Vehicle) IntegrationChargeEntry {
	out := IntegrationChargeEntry{
		ID: c.ID, Date: c.Date.Format("2006-01-02"), Energy: c.KwhAdded, CostToReview: c.CostToReview,
		Synchronised: !c.IsManual, Notes: c.Notes,
	}
	if c.Cost != nil {
		a := amountOf(*c.Cost, vehicle.Currency)
		out.Cost = &a
	}
	if c.Odometer != nil {
		d := kilometres(*c.Odometer)
		out.Odometer = &d
	}
	return out
}
