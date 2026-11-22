package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

// errTollInvoiceChanged is the answer when the invoice was saved after the assistant read it.
var errTollInvoiceChanged = mcpUserError("the toll invoice was changed since it was read: read it again with get_tolls_report, show the owner what it holds now, and redo the change")

// IntegrationTollInvoice is the invoice of one month.
type IntegrationTollInvoice struct {
	Month     string            `json:"month" jsonschema:"the month the invoice covers, YYYY-MM, in the account timezone"`
	Version   int64             `json:"version" jsonschema:"give it back as expected_version to change this invoice with set_toll_invoice"`
	Total     IntegrationAmount `json:"total" jsonschema:"what the operator billed for the month: the month counts this as its tolls in every monthly and global figure"`
	TripTolls IntegrationAmount `json:"trip_tolls" jsonschema:"what the tolls entered or estimated on the trips of the month add up to; the trips, the carpools and the trip groups keep them"`
	Gap       IntegrationAmount `json:"gap" jsonschema:"total minus trip_tolls: positive when the invoice is higher than the tolls recorded on trips (tolls not attached to a trip), negative when it is lower"`
	Note      *string           `json:"note" jsonschema:"the owner's note on the invoice, user text; null when there is none"`
}

func tollInvoiceOf(t models.TollInvoice, currency string) IntegrationTollInvoice {
	return IntegrationTollInvoice{
		Month: t.Month, Version: t.Version, Total: amountOf(t.Total, currency), TripTolls: amountOf(t.TripTolls, currency),
		Gap: amountOf(t.Gap, currency), Note: t.Note,
	}
}

type mcpSetTollInvoiceInput struct {
	VehicleID       string  `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Month           string  `json:"month" jsonschema:"the month the invoice covers, YYYY-MM; not a month that has not started"`
	Total           string  `json:"total" jsonschema:"the total billed for the month, a decimal string in the vehicle's currency such as \"148.30\"; \"0\" says the month had no toll"`
	Note            *string `json:"note,omitempty" jsonschema:"a note on the invoice, such as its number; left out, the note is removed"`
	ExpectedVersion *int64  `json:"expected_version,omitempty" jsonschema:"required to replace an invoice the month already has, as get_tolls_report gave it: the call is refused when the invoice was saved since. Left out to record the first invoice of a month"`
	DryRun          bool    `json:"dry_run,omitempty" jsonschema:"true to see the invoice and the gap it would give, without recording anything"`
}

// IntegrationTollInvoiceSaved is the answer of set_toll_invoice.
type IntegrationTollInvoiceSaved struct {
	Saved    bool                    `json:"saved" jsonschema:"true when the invoice was recorded; false for a dry run, which records nothing"`
	Previous *IntegrationTollInvoice `json:"previous" jsonschema:"the invoice the month had before, null when this is the first one"`
	Invoice  IntegrationTollInvoice  `json:"invoice" jsonschema:"the invoice as it is, or would be on a dry run"`
}

func (h *MCPHandler) addTollInvoiceTools() {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "set_toll_invoice",
		Title: "Record a monthly toll invoice",
		Description: "Records what the motorway operator billed a vehicle in a month. From then on that month counts this total as its tolls in every monthly and global figure, in place of the individual tolls entered or estimated on its trips; the trips, the carpools and the trip groups keep their own tolls, and parking is not affected. " +
			"The answer gives the gap between the invoice and the tolls recorded on trips: positive means tolls that no trip carries, negative means the trips recorded more than was billed. One invoice per vehicle and month: giving a month that already has one replaces it, and needs its version as expected_version (get_tolls_report lists the invoices). An invoice cannot be deleted here. " +
			"This overwrites the owner's figures: call it with dry_run true first, show the owner the gap, and record only once they agree. " +
			"The note is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpSetTollInvoiceInput) (*mcp.CallToolResult, IntegrationTollInvoiceSaved, error) {
		if !mcpCanWrite(req) {
			return nil, IntegrationTollInvoiceSaved{}, errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")
		}
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationTollInvoiceSaved{}, mcpFailure(ctx, "set_toll_invoice", err, "Could not record the toll invoice")
		}
		out, err := h.read.SetTollInvoice(ctx, mcpUserID(req), vehicleID, in)
		if err != nil {
			return nil, IntegrationTollInvoiceSaved{}, mcpFailure(ctx, "set_toll_invoice", err, "Could not record the toll invoice")
		}
		return nil, *out, nil
	})
}

// SetTollInvoice records or replaces the invoice of a month, validated as the toll invoice API is. With in.DryRun
// nothing is stored.
func (h *IntegrationReadHandler) SetTollInvoice(ctx context.Context, userID, vehicleID string, in mcpSetTollInvoiceInput) (*IntegrationTollInvoiceSaved, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a toll invoice cannot be recorded on it")
	}
	loc := h.dateLocation()
	start, end, err := tollInvoiceMonthBounds(in.Month, loc)
	if err != nil {
		return nil, mcpUserError("month must be YYYY-MM, such as \"2026-03\"")
	}
	if start.After(time.Now().In(loc)) {
		return nil, mcpUserError("a month that has not started cannot have an invoice")
	}
	total, err := expenseAmountOf(in.Total, "total")
	if err != nil {
		return nil, err
	}
	var note *string
	if in.Note != nil {
		if t := strings.TrimSpace(*in.Note); t != "" {
			note = &t
		}
	}
	key := start.Format("2006-01")

	var previous *IntegrationTollInvoice
	existing, err := h.repo.GetTollInvoice(ctx, vehicle.ID, key)
	switch {
	case errors.Is(err, database.ErrNotFound):
		if in.ExpectedVersion != nil {
			return nil, mcpUserError(fmt.Sprintf("%s has no invoice to replace: leave expected_version out to record the first one", key))
		}
	case err != nil:
		return nil, err
	default:
		if in.ExpectedVersion == nil {
			return nil, mcpUserError(fmt.Sprintf("%s already has an invoice (version %d, total %s): give that version as expected_version to replace it", key, existing.Version, existing.Total))
		}
		if *in.ExpectedVersion != existing.Version {
			return nil, errTollInvoiceChanged
		}
		p := tollInvoiceOf(*existing, vehicle.Currency)
		previous = &p
	}

	if in.DryRun {
		tripTolls, err := h.repo.TripTollsInPeriod(ctx, vehicle.ID, start, end)
		if err != nil {
			return nil, err
		}
		version := int64(1)
		if existing != nil {
			version = existing.Version + 1
		}
		preview := models.TollInvoice{Month: key, Version: version, Total: total, TripTolls: tripTolls, Gap: total - tripTolls, Note: note}
		return &IntegrationTollInvoiceSaved{Previous: previous, Invoice: tollInvoiceOf(preview, vehicle.Currency)}, nil
	}
	saved, err := h.repo.SaveTollInvoice(ctx, vehicle.ID, key, start, end, total, note, in.ExpectedVersion)
	if errors.Is(err, database.ErrStaleWrite) {
		return nil, errTollInvoiceChanged
	}
	if err != nil {
		return nil, err
	}
	return &IntegrationTollInvoiceSaved{Saved: true, Previous: previous, Invoice: tollInvoiceOf(*saved, vehicle.Currency)}, nil
}
