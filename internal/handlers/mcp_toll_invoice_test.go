package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestSetTollInvoiceRecordsReplacesAndReportsTheGap(t *testing.T) {
	a := newCarpoolAPI(t, "settollinvoice")
	ctx := context.Background()
	h := writeHandler(a)
	day := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	for _, cents := range []int64{1460, 900} {
		e := &models.DriveExpense{VehicleID: a.vid, Type: "TOLL", Amount: money.Cents(cents), Currency: "EUR", Date: day, Source: models.ExpenseSourceManual}
		if err := a.repo.SaveDriveExpense(ctx, e, nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	one, stale := int64(1), int64(9)

	dry, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-09", Total: "30.00", DryRun: true})
	if err != nil || dry.Saved || dry.Previous != nil || dry.Invoice.TripTolls.Amount != "23.60" || dry.Invoice.Gap.Amount != "6.40" {
		t.Fatalf("a dry run shows the gap and records nothing: %+v %v", dry, err)
	}
	if list, _ := a.repo.ListTollInvoices(ctx, a.vid); len(list) != 0 {
		t.Fatalf("a dry run recorded: %+v", list)
	}

	note := " Facture 0926 "
	saved, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-09", Total: "30.00", Note: &note})
	if err != nil || !saved.Saved || saved.Invoice.Version != 1 || *saved.Invoice.Note != "Facture 0926" {
		t.Fatalf("first invoice: %+v %v", saved, err)
	}

	if _, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-09", Total: "10"}); err == nil || !strings.Contains(err.Error(), "expected_version") {
		t.Errorf("replacing without the version read is refused: %v", err)
	}
	if _, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-09", Total: "10", ExpectedVersion: &stale}); err != errTollInvoiceChanged {
		t.Errorf("a stale version: %v", err)
	}
	if _, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-08", Total: "10", ExpectedVersion: &one}); err == nil || !strings.Contains(err.Error(), "no invoice to replace") {
		t.Errorf("a version on a month with no invoice: %v", err)
	}

	replaced, err := h.SetTollInvoice(ctx, a.userID, a.vid, mcpSetTollInvoiceInput{Month: "2026-09", Total: "20", ExpectedVersion: &one})
	if err != nil || replaced.Previous == nil || replaced.Previous.Total.Amount != "30.00" || replaced.Invoice.Version != 2 ||
		replaced.Invoice.Gap.Amount != "-3.60" || replaced.Invoice.Note != nil {
		t.Fatalf("replace: %+v %v", replaced, err)
	}

	for name, in := range map[string]mcpSetTollInvoiceInput{
		"bad month":      {Month: "09/2026", Total: "1"},
		"future month":   {Month: "2999-01", Total: "1"},
		"negative total": {Month: "2026-08", Total: "-1"},
		"not a number":   {Month: "2026-08", Total: "lots"},
	} {
		if _, err := h.SetTollInvoice(ctx, a.userID, a.vid, in); err == nil {
			t.Errorf("%s is refused", name)
		}
	}
	if _, err := h.SetTollInvoice(ctx, a.viewerID, a.vid, mcpSetTollInvoiceInput{Month: "2026-08", Total: "1"}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a viewer cannot record an invoice: %v", err)
	}

	report, err := h.TollsReport(ctx, a.userID, a.vid, "2026-09-01", "2026-09-30", 0)
	if err != nil || len(report.Invoices) != 1 || report.Invoices[0].Total.Amount != "20.00" || report.Total.Amount != "23.60" {
		t.Fatalf("the report lists the invoice next to the individual tolls it does not replace: %+v %v", report, err)
	}
	for _, c := range []struct{ from, to string }{{"2026-10-01", ""}, {"", "2026-08-31"}} {
		other, err := h.TollsReport(ctx, a.userID, a.vid, c.from, c.to, 0)
		if err != nil || len(other.Invoices) != 0 {
			t.Errorf("an invoice outside the period is left out (%v): %+v %v", c, other.Invoices, err)
		}
	}
	all, _ := h.TollsReport(ctx, a.userID, a.vid, "", "", 0)
	if len(all.Invoices) != 1 {
		t.Errorf("no period lists every invoice: %+v", all.Invoices)
	}
}
