package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpDeleteRecordInput struct {
	VehicleID       string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Kind            string `json:"kind" jsonschema:"expense, fuel_fill_up, charge, odometer_reading, toll_invoice or carpool: expense, fuel_fill_up and charge as list_expenses gives them (a charge must be a manual one); odometer_reading as list_odometer_readings gives it; toll_invoice by its month (YYYY-MM) as id; carpool as get_carpool gives it"`
	ID              string `json:"id" jsonschema:"the id of the entry, as the tool that lists or recorded it gives it; the month YYYY-MM for a toll_invoice"`
	ExpectedVersion *int64 `json:"expected_version,omitempty" jsonschema:"the version the dry run showed; required to delete, and the call is refused when the entry was saved since"`
	DryRun          bool   `json:"dry_run,omitempty" jsonschema:"true to see the entry and its version without deleting anything"`
}

// IntegrationRecordSummary is a record as delete_record shows it.
type IntegrationRecordSummary struct {
	Kind    string             `json:"kind" jsonschema:"expense, fuel_fill_up, charge, odometer_reading, toll_invoice or carpool"`
	ID      string             `json:"id" jsonschema:"id of the entry"`
	Version int64              `json:"version" jsonschema:"to give back as expected_version"`
	Date    string             `json:"date" jsonschema:"YYYY-MM-DD"`
	Amount  *IntegrationAmount `json:"amount" jsonschema:"null when the cost is unknown"`
	Note    *string            `json:"note" jsonschema:"user text; null when none"`
}

// IntegrationRecordDeleted is the answer of delete_record.
type IntegrationRecordDeleted struct {
	Deleted bool                     `json:"deleted" jsonschema:"false for a dry run"`
	Record  IntegrationRecordSummary `json:"record" jsonschema:"the entry as it was"`
}

func (h *MCPHandler) addDeleteWriteTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "delete_record",
		Title: "Delete an expense, a fill-up or a manual charge",
		Description: "Deletes one thing the owner recorded on a vehicle: an expense (service, insurance, tax...), a fuel fill-up or a manual charge (found with list_expenses), an odometer reading they typed (list_odometer_readings), a monthly toll invoice (its tolls count again) or a carpool (its drives stay). Charges synchronised from TeslaMate or Home Assistant, trips and tires cannot be deleted here. " +
			"A deletion cannot be undone. Always call it with dry_run true first: it shows the entry and its version. Show it to the owner, and delete only once they agree, with that version as expected_version; the call is refused when the entry was changed since. " +
			"Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, "Could not delete the entry", func(in mcpDeleteRecordInput) string { return in.VehicleID }, h.read.DeleteRecord)
}

// DeleteRecord deletes an expense, a fill-up or a manual charge at the version the caller read. With in.DryRun
// nothing is deleted and the version is not required.
func (h *IntegrationReadHandler) DeleteRecord(ctx context.Context, userID, vehicleID string, in mcpDeleteRecordInput) (*IntegrationRecordDeleted, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: an entry cannot be deleted from it")
	}
	switch in.Kind {
	case database.RecordExpense, database.RecordFuelFill, database.RecordCharge, database.RecordOdometerReading:
	case kindTollInvoice, kindCarpool:
		return h.deleteLinkedRecord(ctx, vehicle, in)
	default:
		return nil, mcpUserError("kind must be expense, fuel_fill_up, charge, odometer_reading, toll_invoice or carpool")
	}
	if in.ID == "" {
		return nil, mcpUserError("id is required: take it from list_expenses")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to delete: call with dry_run true to get it")
	}
	record, err := h.repo.FindRecord(ctx, in.Kind, vehicle.ID, in.ID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("no such entry on this vehicle (a charge must be a manual one): nothing was deleted")
	}
	if err != nil {
		return nil, err
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != record.Version {
		return nil, errRecordChanged
	}
	if !in.DryRun {
		err := h.repo.DeleteRecordAtVersion(ctx, in.Kind, vehicle.ID, in.ID, *in.ExpectedVersion)
		if errors.Is(err, database.ErrStaleWrite) {
			return nil, errRecordChanged
		}
		if errors.Is(err, database.ErrNotFound) {
			return nil, mcpUserError("the entry was already deleted: nothing was deleted")
		}
		if err != nil {
			return nil, err
		}
	}
	return &IntegrationRecordDeleted{Deleted: !in.DryRun, Record: recordSummaryOf(record, vehicle.Currency)}, nil
}

var errRecordChanged = mcpUserError("the entry changed since it was read: call delete_record again with dry_run true to see its current version, then show the owner again")

func recordSummaryOf(r *database.RecordSummary, currency string) IntegrationRecordSummary {
	out := IntegrationRecordSummary{Kind: r.Kind, ID: r.ID, Version: r.Version, Date: r.Date.Format("2006-01-02"), Note: r.Note}
	if r.Amount != nil {
		a := amountOf(*r.Amount, currency)
		out.Amount = &a
	}
	return out
}

const (
	kindTollInvoice = "toll_invoice"
	kindCarpool     = "carpool"
)

// deleteLinkedRecord deletes what is not a row of the plain record tables: a monthly toll invoice (found by its month)
// or a carpool.
func (h *IntegrationReadHandler) deleteLinkedRecord(ctx context.Context, vehicle *models.Vehicle, in mcpDeleteRecordInput) (*IntegrationRecordDeleted, error) {
	if in.ID == "" {
		return nil, mcpUserError("id is required: the month YYYY-MM for a toll invoice, the carpool id of get_carpool for a carpool")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to delete: call with dry_run true to get it")
	}
	var record *database.RecordSummary
	var del func() error
	switch in.Kind {
	case kindTollInvoice:
		invoice, err := h.repo.GetTollInvoice(ctx, vehicle.ID, in.ID)
		if errors.Is(err, database.ErrNotFound) {
			return nil, mcpUserError("that month has no toll invoice: nothing was deleted")
		}
		if err != nil {
			return nil, err
		}
		total := invoice.Total
		start, _ := time.Parse("2006-01", invoice.Month)
		record = &database.RecordSummary{Kind: in.Kind, ID: invoice.Month, Version: invoice.Version, Date: start, Amount: &total, Note: invoice.Note}
		del = func() error { return h.repo.DeleteTollInvoice(ctx, vehicle.ID, invoice.Month, in.ExpectedVersion) }
	default:
		if !vehicleIDPattern.MatchString(in.ID) {
			return nil, mcpUserError("id is not a carpool id: use the id of get_carpool")
		}
		trip, err := h.repo.GetCarpoolTrip(ctx, in.ID, vehicle.ID)
		if errors.Is(err, database.ErrNotFound) {
			return nil, mcpUserError("no such carpool on this vehicle: nothing was deleted")
		}
		if err != nil {
			return nil, err
		}
		total, title := trip.TotalCost, trip.Title
		record = &database.RecordSummary{Kind: in.Kind, ID: trip.ID, Version: trip.Version, Date: trip.Date, Amount: &total, Note: &title}
		del = func() error {
			if *in.ExpectedVersion != trip.Version {
				return database.ErrStaleWrite
			}
			return h.repo.DeleteCarpoolTrip(ctx, trip.ID, vehicle.ID)
		}
	}
	if in.ExpectedVersion != nil && *in.ExpectedVersion != record.Version {
		return nil, errRecordChanged
	}
	if !in.DryRun {
		err := del()
		if errors.Is(err, database.ErrStaleWrite) {
			return nil, errRecordChanged
		}
		if errors.Is(err, database.ErrNotFound) {
			return nil, mcpUserError("it was already deleted: nothing was deleted")
		}
		if err != nil {
			return nil, err
		}
	}
	return &IntegrationRecordDeleted{Deleted: !in.DryRun, Record: recordSummaryOf(record, vehicle.Currency)}, nil
}
