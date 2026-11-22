package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func monthOf(t *testing.T, svc *TCOService, vehicleID, month string) MonthlyCost {
	t.Helper()
	timeline, err := svc.monthlyTimeline(context.Background(), vehicleID)
	if err != nil {
		t.Fatal(err)
	}
	for _, mc := range timeline {
		if mc.Month == month {
			return mc
		}
	}
	t.Fatalf("month %s missing from the timeline", month)
	return MonthlyCost{}
}

func saveInvoice(t *testing.T, repo *database.Repository, loc *time.Location, vehicleID, month string, total money.Cents) *models.TollInvoice {
	t.Helper()
	start, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		t.Fatal(err)
	}
	inv, err := repo.SaveTollInvoice(context.Background(), vehicleID, month, start, start.AddDate(0, 1, 0), total, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func TestIntegrationTollInvoiceReplacesTheTollsOfItsMonthOnly(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "toll-invoice@example.com")
	svc, _ := countingTCOService(t)
	loc := svc.location()

	expense := func(kind string, cents int64, date time.Time) {
		t.Helper()
		e := &models.DriveExpense{VehicleID: v.ID, Type: kind, Amount: money.Cents(cents), Currency: v.Currency, Date: date, Source: models.ExpenseSourceManual}
		if err := repo.SaveDriveExpense(ctx, e, nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	expense("TOLL", 1000, time.Date(2026, 3, 5, 9, 0, 0, 0, loc))
	expense("TOLL", 500, time.Date(2026, 3, 20, 9, 0, 0, 0, loc))
	expense("PARKING", 300, time.Date(2026, 3, 20, 9, 0, 0, 0, loc))
	expense("TOLL", 700, time.Date(2026, 2, 10, 9, 0, 0, 0, loc))
	expense("TOLL", 200, time.Date(2026, 4, 1, 0, 0, 0, 0, loc)) // first instant of April
	expense("TOLL", 900, time.Date(2026, 3, 31, 23, 59, 59, 0, loc))

	if got := monthOf(t, svc, v.ID, "2026-03").Tolls; got != 2400+300 {
		t.Fatalf("without an invoice the individual tolls and the parking count: %d", got)
	}

	inv := saveInvoice(t, repo, loc, v.ID, "2026-03", 5000)
	if inv.TripTolls != 2400 || inv.Gap != 2600 || inv.Version != 1 {
		t.Fatalf("the invoice reports the tolls it replaces and the gap: %+v", inv)
	}
	if got := monthOf(t, svc, v.ID, "2026-03").Tolls; got != 5000+300 {
		t.Errorf("March counts the invoice and keeps the parking: %d", got)
	}
	if got := monthOf(t, svc, v.ID, "2026-02").Tolls; got != 700 {
		t.Errorf("another month is untouched: %d", got)
	}
	if got := monthOf(t, svc, v.ID, "2026-04").Tolls; got != 200 {
		t.Errorf("a toll at the first instant of the next month stays in it: %d", got)
	}

	sum, err := svc.ComputeVehicleTCO(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TollsCost != 700+5000+300+200 {
		t.Errorf("the global tolls line follows the monthly one: %d", sum.TollsCost)
	}

	// An invoice below the tolls entered is a negative gap, and the month still counts the invoice.
	low := saveInvoice(t, repo, loc, v.ID, "2026-03", 1000)
	if low.Gap != -1400 || low.Version != 2 {
		t.Errorf("a lower invoice: %+v", low)
	}
	if got := monthOf(t, svc, v.ID, "2026-03").Tolls; got != 1000+300 {
		t.Errorf("March after the correction: %d", got)
	}

	// 0 says the month had no toll.
	saveInvoice(t, repo, loc, v.ID, "2026-03", 0)
	if got := monthOf(t, svc, v.ID, "2026-03").Tolls; got != 300 {
		t.Errorf("an invoice of 0 removes the individual tolls: %d", got)
	}

	if err := repo.DeleteTollInvoice(ctx, v.ID, "2026-03", nil); err != nil {
		t.Fatal(err)
	}
	if got := monthOf(t, svc, v.ID, "2026-03").Tolls; got != 2700 {
		t.Errorf("deleting the invoice restores the individual tolls: %d", got)
	}
	if err := repo.DeleteTollInvoice(ctx, v.ID, "2026-03", nil); !errors.Is(err, database.ErrNotFound) {
		t.Errorf("deleting twice: %v", err)
	}
}

func TestIntegrationTollInvoiceMonthFollowsTheReportingTimezone(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "toll-invoice-tz@example.com")
	db := repo
	_ = db
	svc, _ := countingTCOService(t)
	svc.timezone = "Pacific/Auckland"
	loc := svc.location()
	if loc.String() != "Pacific/Auckland" {
		t.Fatalf("location %s", loc)
	}
	// 2026-05-31 20:00 UTC is already June 1st in Auckland.
	e := &models.DriveExpense{VehicleID: v.ID, Type: "TOLL", Amount: 400, Currency: v.Currency,
		Date: time.Date(2026, 5, 31, 20, 0, 0, 0, time.UTC), Source: models.ExpenseSourceManual}
	if err := repo.SaveDriveExpense(ctx, e, nil, ""); err != nil {
		t.Fatal(err)
	}
	saveInvoice(t, repo, loc, v.ID, "2026-06", 900)
	if got := monthOf(t, svc, v.ID, "2026-06").Tolls; got != 900 {
		t.Errorf("the toll of June 1st local time is replaced by June's invoice: %d", got)
	}
}

func TestIntegrationTollInvoiceVersionGuardsConcurrentEdits(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "toll-invoice-version@example.com")
	loc := time.UTC
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	one, two := int64(1), int64(2)

	if _, err := repo.SaveTollInvoice(ctx, v.ID, "2026-03", start, end, 100, nil, &one); !errors.Is(err, database.ErrNotFound) {
		t.Errorf("a version on a month that has no invoice: %v", err)
	}
	if _, err := repo.SaveTollInvoice(ctx, v.ID, "2026-03", start, end, 100, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveTollInvoice(ctx, v.ID, "2026-03", start, end, 200, nil, &one); err != nil {
		t.Fatalf("the version that was read: %v", err)
	}
	if _, err := repo.SaveTollInvoice(ctx, v.ID, "2026-03", start, end, 300, nil, &one); !errors.Is(err, database.ErrStaleWrite) {
		t.Errorf("a stale save: %v", err)
	}
	if err := repo.DeleteTollInvoice(ctx, v.ID, "2026-03", &one); !errors.Is(err, database.ErrStaleWrite) {
		t.Errorf("a stale delete: %v", err)
	}
	if err := repo.DeleteTollInvoice(ctx, v.ID, "2026-03", &two); err != nil {
		t.Errorf("a delete with the current version: %v", err)
	}
	if err := repo.DeleteTollInvoice(ctx, v.ID, "2026-03", &two); !errors.Is(err, database.ErrNotFound) {
		t.Errorf("a delete of a missing invoice: %v", err)
	}
}
