package services

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

var alertNow = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

// series builds old entries (outside the alert window) with the given values, then the entry under test.
func series(values []float64, last float64, lastAgeDays int) []alertEntry {
	out := make([]alertEntry, 0, len(values)+1)
	for i, v := range values {
		out = append(out, alertEntry{id: string(rune('a' + i)), at: alertNow.AddDate(0, 0, -100-i), value: v})
	}
	return append(out, alertEntry{id: "last", at: alertNow.AddDate(0, 0, -lastAgeDays), value: last})
}

func TestAmountOutlierNeedsMoreThanFiveTimesTheMedianOfFiveOthers(t *testing.T) {
	usual := []float64{4, 5, 6, 5, 5}
	for _, c := range []struct {
		name    string
		entries []alertEntry
		want    bool
	}{
		{"just over five times", series(usual, 25.5, 3), true},
		{"exactly five times is not reported", series(usual, 25, 3), false},
		{"the usual amount", series(usual, 6, 3), false},
		{"fewer than five comparable records", series(usual[:4], 100, 3), false},
		{"exactly five comparable records", series(usual, 100, 3), true},
		{"older than the alert window", series(usual, 100, alertRecentDays+1), false},
		{"inside the alert window", series(usual, 100, alertRecentDays), true},
		{"a free entry is never an outlier", series(usual, 0, 3), false},
	} {
		got := amountOutlier(c.entries, alertNow)
		if (got != nil) != c.want {
			t.Errorf("%s: reported = %v, want %v", c.name, got != nil, c.want)
		}
		if got != nil && (got.entry.id != "last" || got.median != 5) {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
}

func TestAmountOutlierKeepsTheMostDeviantRecentEntry(t *testing.T) {
	entries := series([]float64{5, 5, 5, 5, 5}, 40, 10)
	entries = append(entries, alertEntry{id: "bigger", at: alertNow.AddDate(0, 0, -5), value: 90})
	got := amountOutlier(entries, alertNow)
	if got == nil || got.entry.id != "bigger" {
		t.Fatalf("got %+v, want the 90 entry", got)
	}
}

func TestPriceOutlierIsMeasuredAgainstTheRangeOfTheOtherPrices(t *testing.T) {
	// Home 0.20 to public 0.55: both ends are usual.
	range_ := []float64{0.20, 0.22, 0.25, 0.40, 0.55}
	for _, c := range []struct {
		name    string
		entries []alertEntry
		want    bool
	}{
		{"a usual price", series(range_, 0.30, 2), false},
		{"a price at the top of the range", series(range_, 0.55, 2), false},
		{"a price just under half past the top", series(range_, 0.82, 2), false},
		{"a price more than half past the top", series(range_, 0.83, 2), true},
		{"a price just above half the lowest", series(range_, 0.10, 2), false},
		{"a price under half the lowest", series(range_, 0.09, 2), true},
		{"a typo by a factor of ten", series(range_, 2.2, 2), true},
		{"fewer than five other prices", series(range_[:4], 5, 2), false},
		{"older than the alert window", series(range_, 2.2, alertRecentDays+1), false},
	} {
		got := priceOutlier(c.entries, alertNow)
		if (got != nil) != c.want {
			t.Errorf("%s: reported = %v, want %v", c.name, got != nil, c.want)
		}
		if got != nil && (got.low != 0.20 || got.high != 0.55) {
			t.Errorf("%s: range %v-%v", c.name, got.low, got.high)
		}
	}
}

func TestIntegrationDataAlertsFollowTheRecords(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "alerts@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{UserID: u.ID, Name: "EV", Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	tco := NewTCOService(db.Pool, "UTC")
	now := time.Now().UTC()

	alerts, err := tco.DataAlerts(ctx, v.ID)
	if err != nil || len(alerts) != 0 {
		t.Fatalf("an empty vehicle has no alert: %+v, %v", alerts, err)
	}

	charge := func(daysAgo int, kwh, cost float64, currency string) {
		t.Helper()
		c := money.FromFloat(cost)
		if err := repo.CreateManualCharge(ctx, &models.ChargeLog{VehicleID: v.ID, Date: now.AddDate(0, 0, -daysAgo), KwhAdded: kwh, Cost: &c, Currency: currency}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 6; i++ {
		charge(100+i*10, 40, 8, "EUR") // 0.20 per kWh
	}
	charge(3, 40, 80, "EUR")  // 2.00 per kWh: a typed 80 instead of 8
	charge(4, 40, 900, "USD") // another currency: no comparable price
	expense := func(kind string, daysAgo int, amount float64, tripGroup bool) {
		t.Helper()
		e := &models.DriveExpense{VehicleID: v.ID, Type: kind, Amount: money.FromFloat(amount), Currency: "EUR", Date: now.AddDate(0, 0, -daysAgo), Source: models.ExpenseSourceManual}
		var ids []string
		var name string
		if tripGroup {
			name = "Holidays"
			d := mustDrive(t, repo, v.ID, daysAgo+1, now.AddDate(0, 0, -daysAgo-1), 1000, 50)
			ids = []string{d.ID}
		}
		if err := repo.SaveDriveExpense(ctx, e, ids, name); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		expense("PARKING", 100+i*10, 3, false)
	}
	expense("PARKING", 2, 60, true) // a trip group is a whole trip's cost: not compared with a single parking fee
	expense("PARKING", 5, 60, false)

	alerts, err = tco.DataAlerts(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("one alert of each kind, got %+v", alerts)
	}
	byKind := map[string]DataAlert{}
	for _, a := range alerts {
		byKind[a.Kind] = a
	}
	if a := byKind[AlertAmountOutlier]; a.Category != "PARKING" || a.Amount != money.FromFloat(60) || a.Median != money.FromFloat(3) {
		t.Errorf("amount outlier: %+v", a)
	}
	if a := byKind[AlertPriceOutlier]; a.Category != "CHARGE" || a.UnitPrice != 2 || a.Low != 0.2 || a.High != 0.2 {
		t.Errorf("price outlier: %+v", a)
	}

	// Correcting the record clears the alert: nothing was stored.
	if _, err := db.Pool.Exec(ctx, `UPDATE charge_logs SET cost = 8 WHERE vehicle_id = $1 AND cost = 80`, v.ID); err != nil {
		t.Fatal(err)
	}
	alerts, err = tco.DataAlerts(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range alerts {
		if a.Kind == AlertPriceOutlier {
			t.Errorf("a corrected price must stop being reported: %+v", a)
		}
	}
}
