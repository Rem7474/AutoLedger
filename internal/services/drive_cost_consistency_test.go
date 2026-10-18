package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// listCost is the cost the drive list shows for a drive: the same steps as the list handler.
func listCost(t *testing.T, ctx context.Context, svc *CarpoolService, repo *database.Repository, vehicleID string, driveID string) DriveCost {
	t.Helper()
	d, err := repo.GetDriveByID(ctx, driveID, vehicleID)
	if err != nil {
		t.Fatal(err)
	}
	drives := []models.Drive{*d}
	rates, err := svc.DriveRates(ctx, vehicleID, drives)
	if err != nil {
		t.Fatal(err)
	}
	insurance, err := svc.AllocateInsuranceCosts(ctx, vehicleID, drives, rates)
	if err != nil {
		t.Fatal(err)
	}
	tolls, err := repo.GetTollExpensesForDrives(ctx, vehicleID, []string{driveID})
	if err != nil {
		t.Fatal(err)
	}
	return ComputeDriveCost(rates[driveID], *d, insurance[driveID], tolls[driveID])
}

func chargeAt(t *testing.T, ctx context.Context, repo *database.Repository, vehicleID string, date time.Time, kwh float64, cost *money.Cents) *models.ChargeLog {
	t.Helper()
	c := &models.ChargeLog{VehicleID: vehicleID, Date: date, KwhAdded: kwh, Cost: cost, Currency: "EUR"}
	if err := repo.CreateManualCharge(ctx, c); err != nil {
		t.Fatal(err)
	}
	return c
}

// A drive has one cost: the drive list, the carpool estimate and the carpool leg saved by a recalculation agree on
// it, and a charge cost entered after the drive (for a charge dated before it) reaches all three.
func TestIntegrationDriveCostIsTheSameInTheListAndTheCarpool(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "cost_unified@example.com")
	svc := NewCarpoolService(db.Pool, repo)

	driveStart := time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC)
	d := mustDrive(t, repo, v.ID, 501, driveStart, 20000, 50)

	// The charge before the drive has no cost yet
	charge := chargeAt(t, ctx, repo, v.ID, driveStart.AddDate(0, 0, -2), 40, nil)

	before := listCost(t, ctx, svc, repo, v.ID, d.ID)
	if before.ElectricityCost != 0 {
		t.Fatalf("a charge without cost must not price the drive, got %v", before.ElectricityCost)
	}

	cost := money.Cents(1200)
	charge.Cost = &cost
	if err := repo.UpdateCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}

	list := listCost(t, ctx, svc, repo, v.ID, d.ID)
	if list.ElectricityCost <= 0 {
		t.Fatalf("the cost entered afterwards must reach the drive, got %+v", list)
	}

	driveID := d.ID
	est, err := svc.EstimateCosts(ctx, v.ID, &driveID, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if est.ElectricityCost != list.ElectricityCost || est.TiresCost != list.TiresCost ||
		est.MaintenanceCost != list.MaintenanceCost || est.InsuranceCost != list.InsuranceCost ||
		est.TotalCost != list.TotalCost {
		t.Fatalf("carpool estimate %+v differs from the drive list %+v", est, list)
	}
}

// The rates a drive is priced with are those in force when it ended: a charge dated after the drive is ignored,
// and each drive of a list gets its own rate whatever the order the batch is asked in.
func TestIntegrationDriveRatesFollowEachDrivesDate(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "drive_rates_dates@example.com")
	svc := NewCarpoolService(db.Pool, repo)

	day := func(n int) time.Time { return time.Date(2026, 3, n, 8, 0, 0, 0, time.UTC) }
	one, three, nine := money.Cents(100), money.Cents(300), money.Cents(900)
	chargeAt(t, ctx, repo, v.ID, day(1), 10, &one)
	chargeAt(t, ctx, repo, v.ID, day(20), 10, &three)
	chargeAt(t, ctx, repo, v.ID, day(28), 10, &nine)

	early := mustDrive(t, repo, v.ID, 601, day(5), 20000, 30)
	late := mustDrive(t, repo, v.ID, 602, day(25), 20030, 30)

	drives := []models.Drive{*late, *early}
	rates, err := svc.DriveRates(ctx, v.ID, drives)
	if err != nil {
		t.Fatal(err)
	}
	// Day 5: only the first charge exists. Day 25: the last charge is the one of day 20 (the one of day 1 is more
	// than 5 days earlier, and the one of day 28 is after the drive).
	if got := rates[early.ID].ElectricityPerKwh; got != 0.1 {
		t.Errorf("early drive rate: got %v, want 0.1", got)
	}
	if got := rates[late.ID].ElectricityPerKwh; got != 0.3 {
		t.Errorf("late drive rate: got %v, want 0.3", got)
	}

	// The batch gives, for each reference, what a single reference gives
	e, l := early.EndTime, late.EndTime
	batch, err := svc.GetVehicleUnitRatesAtMany(ctx, v.ID, []*time.Time{&l, nil, &e})
	if err != nil {
		t.Fatal(err)
	}
	for i, ref := range []*time.Time{&l, nil, &e} {
		single, err := svc.GetVehicleUnitRatesAt(ctx, v.ID, ref)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(batch[i], single) {
			t.Errorf("reference %d: batch %+v differs from single %+v", i, *batch[i], *single)
		}
	}
	// Today the latest charge is more than 30 days old: the rate is the average of the whole history
	if got := batch[1].ElectricityPerKwh; batch[1].ElectricitySource != RateSourceHistory || got < 0.433 || got > 0.434 {
		t.Errorf("current rate: got %v (%s), want the history average 0.433", got, batch[1].ElectricitySource)
	}
}
