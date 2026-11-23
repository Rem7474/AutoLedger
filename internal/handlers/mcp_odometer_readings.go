package handlers

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpListOdometerReadingsInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
}

// IntegrationOdometerReadingEntry is an odometer reading with the version to quote when changing or deleting it.
type IntegrationOdometerReadingEntry struct {
	ID       string              `json:"id" jsonschema:"the id of the reading"`
	Date     string              `json:"date" jsonschema:"the day of the reading, YYYY-MM-DD"`
	Odometer IntegrationDistance `json:"odometer" jsonschema:"what the odometer showed"`
	Notes    *string             `json:"notes" jsonschema:"the note on the reading, user text; null when none"`
	Version  int64               `json:"version" jsonschema:"to give back as expected_version"`
}

// IntegrationOdometerReadings is the answer of list_odometer_readings.
type IntegrationOdometerReadings struct {
	Readings []IntegrationOdometerReadingEntry `json:"readings" jsonschema:"the readings the owner typed, oldest first; the ones an integration reports are not listed"`
}

type mcpUpdateOdometerInput struct {
	VehicleID       string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	ReadingID       string   `json:"reading_id" jsonschema:"the id of the reading, as list_odometer_readings or record_odometer gives it"`
	ExpectedVersion *int64   `json:"expected_version,omitempty" jsonschema:"the version the dry run showed; required to save, and the call is refused when the reading was saved since"`
	Date            *string  `json:"date,omitempty" jsonschema:"the day of the reading, YYYY-MM-DD"`
	OdometerKm      *float64 `json:"odometer_km,omitempty" jsonschema:"what the odometer showed, in kilometres, between 0 and 2000000"`
	Notes           *string  `json:"notes,omitempty" jsonschema:"a free note, 500 characters at most; an empty string removes it"`
	DryRun          bool     `json:"dry_run,omitempty" jsonschema:"true to check the change against the other readings and see it, without saving anything"`
}

// IntegrationOdometerUpdated is the answer of update_odometer_reading.
type IntegrationOdometerUpdated struct {
	Updated bool                            `json:"updated" jsonschema:"false for a dry run"`
	Version int64                           `json:"version" jsonschema:"the version now"`
	Before  IntegrationOdometerReadingEntry `json:"before" jsonschema:"the reading as it was"`
	After   IntegrationOdometerReadingEntry `json:"after" jsonschema:"the reading as it is, or would be, saved"`
}

func (h *MCPHandler) addOdometerReadingTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "list_odometer_readings",
		Title: "Odometer readings the owner typed",
		Description: "The dated odometer readings the owner typed for a vehicle, with their ids and versions, to change one with update_odometer_reading or delete it with delete_record (kind odometer_reading). " +
			"The mileage an integration reports is not listed: it cannot be changed here.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpListOdometerReadingsInput) (*mcp.CallToolResult, IntegrationOdometerReadings, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationOdometerReadings{}, mcpFailure(ctx, "list_odometer_readings", err, "Could not list the odometer readings")
		}
		checkpoints, err := h.read.repo.ListOdometerCheckpoints(ctx, vehicleID)
		if err != nil {
			return nil, IntegrationOdometerReadings{}, mcpFailure(ctx, "list_odometer_readings", err, "Could not list the odometer readings")
		}
		out := IntegrationOdometerReadings{Readings: []IntegrationOdometerReadingEntry{}}
		for i := range checkpoints {
			if checkpoints[i].Source == "MANUAL" {
				out.Readings = append(out.Readings, readingEntryOf(&checkpoints[i]))
			}
		}
		return nil, out, nil
	})
}

func readingEntryOf(c *models.OdometerCheckpoint) IntegrationOdometerReadingEntry {
	return IntegrationOdometerReadingEntry{ID: c.ID, Date: c.Date.Format("2006-01-02"), Odometer: kilometres(c.Odometer), Notes: c.Notes, Version: c.Version}
}

func (h *MCPHandler) addOdometerUpdateTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "update_odometer_reading",
		Title: "Change an odometer reading",
		Description: "Changes an odometer reading the owner typed (found with list_odometer_readings): its day, mileage or note. The change is checked against the other readings and a mileage that contradicts them is refused. " +
			"Call it with dry_run true first: it shows the reading and its version. Show the change to the owner, and save only once they agree, with that version as expected_version. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, "Could not change the odometer reading", func(in mcpUpdateOdometerInput) string { return in.VehicleID }, h.read.UpdateOdometerReading)
}

// UpdateOdometerReading changes the given fields of a typed reading at the version the caller read.
func (h *IntegrationReadHandler) UpdateOdometerReading(ctx context.Context, userID, vehicleID string, in mcpUpdateOdometerInput) (*IntegrationOdometerUpdated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: an odometer reading cannot be changed on it")
	}
	if in.ReadingID == "" {
		return nil, mcpUserError("reading_id is required: take it from list_odometer_readings")
	}
	if in.Date == nil && in.OdometerKm == nil && in.Notes == nil {
		return nil, mcpUserError("nothing to change: give at least one of date, odometer_km, notes")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to save: call with dry_run true to get it")
	}
	if in.Notes != nil && len(*in.Notes) > 500 {
		return nil, mcpUserError("notes is over 500 characters")
	}
	checkpoints, err := h.repo.ListOdometerCheckpoints(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	var current *models.OdometerCheckpoint
	for i := range checkpoints {
		if checkpoints[i].ID == in.ReadingID && checkpoints[i].Source == "MANUAL" {
			current = &checkpoints[i]
		}
	}
	if current == nil {
		return nil, mcpUserError("no such reading the owner typed on this vehicle: nothing was changed")
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != current.Version {
		return nil, errRecordChanged
	}
	date, odometer, note := current.Date.Format("2006-01-02"), current.Odometer, current.Notes
	if in.Date != nil {
		date = *in.Date
	}
	if in.OdometerKm != nil {
		odometer = *in.OdometerKm
	}
	if in.Notes != nil {
		note = in.Notes
	}
	day, notes, err := validateCheckpoint(date, odometer, note)
	if err != nil {
		return nil, err
	}
	points, err := h.repo.ListManualOdometerPoints(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	if err := checkOdometerOrder(points, current.ID, day, odometer); err != nil {
		return nil, err
	}
	next := &models.OdometerCheckpoint{ID: current.ID, VehicleID: vehicle.ID, Date: day, Odometer: odometer, Notes: notes, Version: current.Version}
	out := &IntegrationOdometerUpdated{Version: current.Version, Before: readingEntryOf(current)}
	if !in.DryRun {
		next.ExpectedVersion = in.ExpectedVersion
		if err := h.repo.UpdateOdometerCheckpoint(ctx, next); errors.Is(err, database.ErrStaleWrite) {
			return nil, errRecordChanged
		} else if err != nil {
			return nil, err
		}
		out.Updated, out.Version = true, next.Version
	}
	out.After = readingEntryOf(next)
	return out, nil
}
