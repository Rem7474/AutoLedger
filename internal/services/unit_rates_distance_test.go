package services

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestOdometerDistanceKm(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d) }
	anchors := []OdometerAnchor{{Date: day(0), Km: 1000}, {Date: day(100), Km: 11000}}

	if got := odometerDistanceKm(anchors, day(50), day(100)); got != 5000 {
		t.Fatalf("interpolated distance = %v, want 5000", got)
	}
	if got := odometerDistanceKm(anchors, time.Time{}, day(100)); got != 10000 {
		t.Fatalf("distance from the first reading = %v, want 10000", got)
	}
	if got := odometerDistanceKm(nil, day(0), day(10)); got != 0 {
		t.Fatalf("no reading must give 0, got %v", got)
	}
	if got := odometerDistanceKm(anchors, day(100), day(100)); got != 0 {
		t.Fatalf("an empty period must give 0, got %v", got)
	}
}

func TestCoveredDistanceKm(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d) }
	anchors := []OdometerAnchor{{Date: day(0), Km: 0, Origin: true}, {Date: day(100), Km: 10000}}

	if got := coveredDistanceKm(900, anchors, day(0), day(100)); got != 10000 {
		t.Fatalf("drives covering a fraction of the odometer must not be the denominator, got %v", got)
	}
	if got := coveredDistanceKm(12000, anchors, day(0), day(100)); got != 12000 {
		t.Fatalf("drives above the odometer estimate must be kept, got %v", got)
	}
	if got := coveredDistanceKm(900, nil, day(0), day(100)); got != 900 {
		t.Fatalf("without readings the drives are the only distance, got %v", got)
	}
}

// A vehicle fed with a few weeks of drives (Home Assistant, manual entry) over years of costs: the per-km rates must
// be spread over the kilometres the odometer shows, not over the few recorded drives.
func TestIntegrationUnitRatesUseOdometerWhenDrivesAreSparse(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "rates-sparse-drives@example.com")
	now := time.Now().UTC()

	if err := repo.CreateOdometerCheckpoint(ctx, &models.OdometerCheckpoint{VehicleID: v.ID, Date: now.AddDate(0, 0, -400), Odometer: 10000}); err != nil {
		t.Fatal(err)
	}
	// Three short drives in the last weeks: 30 km recorded against about 10 000 km driven.
	for i := 0; i < 3; i++ {
		mustDrive(t, repo, v.ID, 9000+i, now.AddDate(0, 0, -10+i), 19970+float64(i*10), 10)
	}
	months := 1
	premium := &models.MaintenanceExpense{VehicleID: v.ID, Category: "INSURANCE", Date: now.AddDate(0, 0, -400), Amount: money.FromFloat(50), Currency: "EUR", IsRecurring: true, RecurrenceIntervalMonths: &months}
	if err := repo.CreateMaintenanceExpense(ctx, premium); err != nil {
		t.Fatal(err)
	}
	vid := v.ID
	tire := &models.Tire{
		VehicleID: &vid, Brand: "B", Model: "M", Dimension: "205/55R16", Season: models.TireSeasonWinter,
		PurchaseDate: now.AddDate(0, 0, -400), PurchasePrice: money.FromFloat(100), CurrentPosition: models.TirePosStorage,
		InitialDepthMm: 8, MinLegalDepthMm: 1.6,
	}
	if err := repo.CreateTire(ctx, tire); err != nil {
		t.Fatal(err)
	}

	rates, err := NewCarpoolService(db.Pool, repo).GetVehicleUnitRatesAt(ctx, v.ID, &now)
	if err != nil {
		t.Fatal(err)
	}
	// 100 EUR of tires over ~10 000 km is 0.01 EUR/km; over the 30 recorded km it would be 3.33.
	if rates.TiresSource != RateSourceHistory || rates.TiresPerKm <= 0 || rates.TiresPerKm > 0.05 {
		t.Fatalf("tires rate = %v (%s), want a few cents at most", rates.TiresPerKm, rates.TiresSource)
	}
	// About 600 EUR of premiums over ~9 000 km is 0.07 EUR/km; over 30 km it would be 20.
	if rates.InsuranceSource != InsuranceSourceRecordedExpenses || rates.InsurancePerKm <= 0 || rates.InsurancePerKm > 0.2 {
		t.Fatalf("insurance rate = %v (%s), want well under 0.2", rates.InsurancePerKm, rates.InsuranceSource)
	}
}
