package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationExpensesListEntriesWithTheirPlaceAndTotalsOfEveryMatch(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "expenses-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "expenses-other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{UserID: owner.ID, Name: "Daily", Currency: "GBP", TeslaMateAuthType: models.AuthModeNone, CurrentOdometer: 1000}
	stranger := &models.Vehicle{UserID: other.ID, Name: "Not mine", Currency: "GBP", TeslaMateAuthType: models.AuthModeNone}
	for _, x := range []*models.Vehicle{v, stranger} {
		if err := repo.CreateVehicle(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	place, err := repo.CreateChargingPlace(ctx, owner.ID, "Home", "HOME")
	if err != nil {
		t.Fatal(err)
	}
	address := "1 Example Street"
	if err := repo.SetChargingPlaceAlias(ctx, owner.ID, address, &place.ID); err != nil {
		t.Fatal(err)
	}
	cost := money.Cents(1250)
	charge := &models.ChargeLog{VehicleID: v.ID, Date: time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), Address: &address, KwhAdded: 40, Cost: &cost, Currency: "GBP"}
	if err := repo.CreateManualCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}
	receipt := &models.ExpenseDocument{UserID: owner.ID, VehicleID: v.ID, Filename: "invoice.pdf", MimeType: "application/pdf", FileSize: 10}
	if err := repo.SaveExpenseDocument(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	service := &models.MaintenanceExpense{VehicleID: v.ID, Category: "MAINTENANCE", Amount: 12000, Currency: "GBP", Date: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Description: "Brake pads", DocumentID: &receipt.ID}
	insurance := &models.MaintenanceExpense{VehicleID: v.ID, Category: "INSURANCE", Amount: 45000, Currency: "GBP", Date: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Description: "Yearly premium"}
	foreign := &models.MaintenanceExpense{VehicleID: stranger.ID, Category: "MAINTENANCE", Amount: 99900, Currency: "GBP", Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Description: "Not yours"}
	for _, m := range []*models.MaintenanceExpense{service, insurance, foreign} {
		if err := repo.CreateMaintenanceExpense(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	h := NewIntegrationReadHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	all, err := h.Expenses(ctx, owner.ID, v.ID, IntegrationExpenseQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if all.Count != 3 || all.Total.Amount != "582.50" || len(all.Expenses) != 3 || all.Expenses[0].Date != "2026-03-10" || all.NextCursor != nil {
		t.Fatalf("every entry of the vehicle, newest first, and none of another account's: %+v", all)
	}
	if all.ByCategory[0].Category != "insurance" || all.ByCategory[0].Amount.Amount != "450.00" || len(all.ByCategory) != 3 {
		t.Errorf("categories, the largest first: %+v", all.ByCategory)
	}
	var charged *IntegrationExpense
	for i := range all.Expenses {
		if all.Expenses[i].Kind == "charge" {
			charged = &all.Expenses[i]
		}
	}
	if charged == nil || charged.Place == nil || *charged.Place != "Home" || charged.PlaceCategory == nil || *charged.PlaceCategory != "home" || charged.Kwh == nil || *charged.Kwh != 40 || charged.Amount.Amount != "12.50" {
		t.Errorf("a charge carries its named place and its energy: %+v", charged)
	}

	receipts := map[string]*bool{}
	for _, e := range all.Expenses {
		receipts[e.Kind+"/"+e.Category] = e.HasReceipt
	}
	if r := receipts["expense/maintenance"]; r == nil || !*r {
		t.Errorf("an expense with an attached receipt says so: %v", r)
	}
	if r := receipts["expense/insurance"]; r == nil || *r {
		t.Errorf("an expense without a receipt says so: %v", r)
	}
	if r := receipts["charge/energy"]; r == nil || *r {
		t.Errorf("a charge without a receipt says so: %v", r)
	}

	page, err := h.Expenses(ctx, owner.ID, v.ID, IntegrationExpenseQuery{Sort: "amount", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Expenses) != 2 || page.Expenses[0].Category != "insurance" || page.NextCursor == nil || page.Count != 3 || page.Total.Amount != "582.50" {
		t.Fatalf("the first page by amount, with totals over the whole set: %+v", page)
	}
	next, err := h.Expenses(ctx, owner.ID, v.ID, IntegrationExpenseQuery{Sort: "amount", Limit: 2, Cursor: *page.NextCursor})
	if err != nil || len(next.Expenses) != 1 || next.Expenses[0].Category != "energy" || next.NextCursor != nil {
		t.Fatalf("the last page: %+v %v", next, err)
	}

	filtered, err := h.Expenses(ctx, owner.ID, v.ID, IntegrationExpenseQuery{From: "2026-03-01", To: "2026-03-31", Categories: []string{"Maintenance", "energy"}, MinAmount: "20"})
	if err != nil || filtered.Count != 1 || filtered.Expenses[0].Category != "maintenance" || filtered.Total.Amount != "120.00" || *filtered.Expenses[0].Note != "Brake pads" {
		t.Fatalf("period, categories and minimum amount: %+v %v", filtered, err)
	}

	if _, err := h.Expenses(ctx, other.ID, v.ID, IntegrationExpenseQuery{}); err == nil {
		t.Error("another account cannot list the entries of this vehicle")
	}
}

func TestIntegrationExpenseQueriesAreRefusedWithAMessageThatSaysWhatToFix(t *testing.T) {
	loc := time.UTC
	for name, q := range map[string]IntegrationExpenseQuery{
		"category": {Categories: []string{"fuel"}},
		"amount":   {MinAmount: "lots"},
		"sort":     {Sort: "name"},
		"cursor":   {Cursor: "zzz"},
		"day":      {From: "March"},
		"period":   {From: "2026-05-01", To: "2026-01-01"},
	} {
		_, err := expenseFilter(q, loc)
		if _, ok := apierror.As(err); !ok {
			t.Errorf("%s: want a user-facing error, got %v", name, err)
		}
	}
	f, err := expenseFilter(IntegrationExpenseQuery{To: "2026-03-31", Limit: 9999}, loc)
	if err != nil || f.Limit != integrationExpensesMax || !f.To.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, loc)) {
		t.Errorf("the last day is included and the page is capped: %+v %v", f, err)
	}
}
