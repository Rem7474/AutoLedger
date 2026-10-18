package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestContinuousCostMonthsFillsTheGapsAndReachesTheCurrentMonth(t *testing.T) {
	got := continuousCostMonths(map[string]money.Cents{"2025-11": 1000, "2026-02": 250}, "2026-04")
	want := []CostHistoryMonth{
		{"2025-11", 1000}, {"2025-12", 0}, {"2026-01", 0}, {"2026-02", 250}, {"2026-03", 0}, {"2026-04", 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestContinuousCostMonthsKeepsARecordedMonthAfterTheCurrentOne(t *testing.T) {
	got := continuousCostMonths(map[string]money.Cents{"2026-05": 900}, "2026-03")
	if len(got) != 1 || got[0].Month != "2026-05" || got[0].Amount != 900 {
		t.Fatalf("got %v", got)
	}
}

func TestContinuousCostMonthsWithoutRecordIsTheCurrentMonth(t *testing.T) {
	got := continuousCostMonths(nil, "2026-04")
	if !reflect.DeepEqual(got, []CostHistoryMonth{{Month: "2026-04"}}) {
		t.Fatalf("got %v", got)
	}
}

func TestIntegrationCostHistoryFollowsTheListsAndStaysScoped(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	alice := mustVehicle(t, repo, "history-alice@example.com")
	bob := mustVehicle(t, repo, "history-bob@example.com")
	svc := NewTCOService(db.Pool, "Europe/Paris")

	// Two days that fall in different months once read in the reporting timezone: 31 Jan 23:30 UTC is 1 Feb in Paris.
	for _, c := range []struct {
		vehicle, typ string
		amount       float64
		at           time.Time
	}{
		{alice.ID, "TOLL", 10, time.Date(2026, 1, 31, 23, 30, 0, 0, time.UTC)},
		{alice.ID, "PARKING", 5, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
		{bob.ID, "TOLL", 99, time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
	} {
		if _, err := db.Pool.Exec(ctx, `INSERT INTO drive_expenses (vehicle_id, type, amount, date) VALUES ($1, $2, $3, $4)`, c.vehicle, c.typ, c.amount, c.at); err != nil {
			t.Fatal(err)
		}
	}

	monthly := 1
	start := time.Now().In(time.UTC).AddDate(0, -2, 0)
	create := func(category, description string, amount float64, recurring bool) {
		t.Helper()
		e := &models.MaintenanceExpense{VehicleID: alice.ID, Category: category, Currency: "EUR", Date: start, Amount: eur(amount), Description: description, AmortizationMode: "NONE"}
		if recurring {
			e.IsRecurring, e.RecurrenceIntervalMonths = true, &monthly
		}
		if err := repo.CreateMaintenanceExpense(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	create("INSURANCE", "Policy", 100, true)
	create("TAX", "Road tax", 40, false)
	create("MAINTENANCE", "Service", 300, false)
	create("ACCESSORY", "Roof box", 50, false)

	tolls, err := svc.CostHistory(ctx, alice.ID, CostHistoryTolls)
	if err != nil {
		t.Fatal(err)
	}
	// The first toll is on 1 February in the reporting timezone, the parking on 10 March
	if tolls.Months[0] != (CostHistoryMonth{"2026-02", eur(10)}) || tolls.Months[1] != (CostHistoryMonth{"2026-03", eur(5)}) {
		t.Fatalf("tolls = %v", tolls.Months)
	}
	var tollTotal money.Cents
	for _, m := range tolls.Months {
		tollTotal += m.Amount
	}
	if tollTotal != eur(15) {
		t.Fatalf("tolls total = %d, want 15 EUR (bob's toll must not count)", tollTotal)
	}

	sum := func(h *CostHistory) (total money.Cents) {
		for _, m := range h.Months {
			total += m.Amount
		}
		return
	}
	fixed, err := svc.CostHistory(ctx, alice.ID, CostHistoryFixed)
	if err != nil {
		t.Fatal(err)
	}
	// A policy due every month since two months ago is three payments, the one-off tax is one
	if got, want := sum(fixed), eur(100*3+40); got != want {
		t.Fatalf("fixed total = %d, want %d", got, want)
	}
	service, err := svc.CostHistory(ctx, alice.ID, CostHistoryService)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sum(service), eur(350); got != want {
		t.Fatalf("service total = %d, want %d (a fixed cost never counts as servicing)", got, want)
	}
	if last := fixed.Months[len(fixed.Months)-1]; last.Month != fixed.CurrentMonth {
		t.Fatalf("history ends on %s, want the current month %s", last.Month, fixed.CurrentMonth)
	}

	empty, err := svc.CostHistory(ctx, bob.ID, CostHistoryService)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Months) != 1 || empty.Months[0].Amount != 0 {
		t.Fatalf("a vehicle without a record has the current month alone, got %v", empty.Months)
	}
	if _, err := svc.CostHistory(ctx, alice.ID, "energy"); !errors.Is(err, ErrUnsupportedCostHistoryKind) {
		t.Fatalf("an unknown kind must be refused, got %v", err)
	}
}
