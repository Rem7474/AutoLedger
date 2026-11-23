package handlers

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A vehicle shared read-only is refused by every writing tool that works on a vehicle's records: the role is checked
// in the service of each tool, so each one needs its own case.
func TestMCPExpenseWritesRefuseAReadOnlyMember(t *testing.T) {
	a := newCarpoolAPI(t, "mcpwrite-viewer")
	ctx := context.Background()
	d := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon", "Paris", 100)
	h := writeHandler(a)

	create := mcpCreateExpenseInput{Category: "subscription", Amount: "6.00", Date: "2026-09-01", Description: "Badge fee"}
	if _, err := h.CreateExpense(ctx, a.viewerID, a.vid, create); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot record an expense: %v", err)
	}
	update := mcpUpdateDriveExpensesInput{Items: []mcpDriveExpenseItem{{DriveID: d.ID, Tolls: ptr("5.00")}}}
	if _, err := h.UpdateDriveExpenses(ctx, a.viewerID, a.vid, update); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot change the tolls of a trip: %v", err)
	}

	if _, err := h.RecordOdometer(ctx, a.viewerID, a.vid, mcpRecordOdometerInput{Date: "2026-09-06", OdometerKm: 20000}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot record an odometer reading: %v", err)
	}

	if _, err := h.AddFuelFill(ctx, a.viewerID, a.vid, mcpAddFuelFillInput{Date: "2026-09-06", Amount: "50.00"}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot record a fill-up: %v", err)
	}

	if _, err := h.AddCharge(ctx, a.viewerID, a.vid, mcpAddChargeInput{Date: "2026-09-06", KwhAdded: 20, Cost: "5.00"}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot record a charge: %v", err)
	}

	if _, err := h.DeleteRecord(ctx, a.viewerID, a.vid, mcpDeleteRecordInput{Kind: "expense", ID: "x", DryRun: true}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("a read-only member cannot delete an entry: %v", err)
	}

	// The owner of the same vehicle is not refused, so the cases above fail on the role and on nothing else
	if _, err := h.CreateExpense(ctx, a.userID, a.vid, mcpCreateExpenseInput{Category: "subscription", Amount: "6.00", Date: "2026-09-01", Description: "Badge fee", DryRun: true}); err != nil {
		t.Errorf("the owner can: %v", err)
	}
}
