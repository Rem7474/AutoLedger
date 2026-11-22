package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/models"
)

type mcpRecordOdometerInput struct {
	VehicleID      string  `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Date           string  `json:"date" jsonschema:"the day of the reading, YYYY-MM-DD"`
	OdometerKm     float64 `json:"odometer_km" jsonschema:"what the odometer showed, in kilometres, between 0 and 2000000"`
	Notes          string  `json:"notes,omitempty" jsonschema:"a free note on the reading, 500 characters at most"`
	DryRun         bool    `json:"dry_run,omitempty" jsonschema:"true to check the reading against the others and see it, without recording anything"`
	IdempotencyKey string  `json:"idempotency_key,omitempty" jsonschema:"a value you make up, 100 characters at most, so that a retry records once: sending the call again with the same key and the same arguments returns the first answer and records nothing more, and a different call with the same key is refused. Without it, a retry may record twice. Not used by a dry run"`
}

// IntegrationOdometerReading is an odometer reading the owner recorded.
type IntegrationOdometerReading struct {
	ID       string              `json:"id" jsonschema:"the id of the reading; empty on a dry run"`
	Date     string              `json:"date" jsonschema:"the day of the reading, YYYY-MM-DD"`
	Odometer IntegrationDistance `json:"odometer" jsonschema:"what the odometer showed"`
	Notes    *string             `json:"notes" jsonschema:"the note on the reading, user text; null when none"`
}

// IntegrationOdometerRecorded is the answer of record_odometer.
type IntegrationOdometerRecorded struct {
	Recorded bool                       `json:"recorded" jsonschema:"true when the reading was recorded; false for a dry run, which records nothing"`
	Reading  IntegrationOdometerReading `json:"reading" jsonschema:"the reading as it is, or would be, recorded"`
}

func (h *MCPHandler) addOdometerWriteTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "record_odometer",
		Title: "Record an odometer reading",
		Description: "Records what the odometer of a vehicle showed on a day, as the odometer form of the app does: it moves the vehicle's current odometer when the reading is the latest, and feeds the estimates of the energy and the distance between readings. " +
			"A reading that contradicts the others (a lower mileage on a later day, or the other way round) is refused with the reading it contradicts. It never changes or deletes a reading. " +
			"This writes to the owner's records: call it with dry_run true first, show the owner the reading, and record only once they agree. Pass an idempotency_key so that a retry after a lost answer records once. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not record the odometer reading", func(in mcpRecordOdometerInput) string { return in.VehicleID }, h.read.RecordOdometer,
		withIdempotencyKey(func(in mcpRecordOdometerInput) string { return in.IdempotencyKey }, func(in mcpRecordOdometerInput) bool { return in.DryRun }))
}

// RecordOdometer records an odometer reading, validated and checked against the other readings as the checkpoint
// routes do. With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) RecordOdometer(ctx context.Context, userID, vehicleID string, in mcpRecordOdometerInput) (*IntegrationOdometerRecorded, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: an odometer reading cannot be recorded on it")
	}
	if len(in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	var note *string
	if in.Notes != "" {
		note = &in.Notes
	}
	date, notes, err := validateCheckpoint(in.Date, in.OdometerKm, note)
	if err != nil {
		return nil, err
	}
	points, err := h.repo.ListManualOdometerPoints(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	if err := checkOdometerOrder(points, "", date, in.OdometerKm); err != nil {
		return nil, err
	}
	checkpoint := &models.OdometerCheckpoint{VehicleID: vehicle.ID, Date: date, Odometer: in.OdometerKm, Notes: notes}
	if !in.DryRun {
		if err := h.repo.CreateOdometerCheckpoint(ctx, checkpoint); err != nil {
			return nil, err
		}
	}
	return &IntegrationOdometerRecorded{
		Recorded: !in.DryRun,
		Reading: IntegrationOdometerReading{
			ID: checkpoint.ID, Date: date.Format("2006-01-02"), Odometer: kilometres(in.OdometerKm), Notes: notes,
		},
	}, nil
}
