package database

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

// A drive entered by hand, with its odometer, carries no power telemetry and an estimated consumption: it gives the
// tire no driving style, so the wear forecast is not adjusted by an invented one.
func TestDrivingTelemetryIgnoresManualDrives(t *testing.T) {
	repo, ctx, _, _, vehicle := vehicleRepoFixture(t, "telemetry-manual")
	now := time.Now().UTC()
	km, start, end, kwh100 := 40.0, 12000.0, 12040.0, 18.0
	manual := &models.Drive{VehicleID: vehicle.ID, StartTime: now, EndTime: now, DistanceKm: &km, StartOdometer: &start, EndOdometer: &end, ConsumptionKwh100km: &kwh100, EnergyEstimated: true}
	if err := repo.CreateManualDrive(ctx, manual); err != nil {
		t.Fatal(err)
	}
	maxOdo := 20000.0
	powerMax, powerMin, consumption, count, err := repo.GetDrivingTelemetryStats(ctx, vehicle.ID, []OdometerRange{{Min: 10000, Max: &maxOdo}})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || powerMax != 0 || powerMin != 0 || consumption != 0 {
		t.Errorf("a manual drive is no telemetry: count %d, power %v/%v, consumption %v", count, powerMax, powerMin, consumption)
	}
}
