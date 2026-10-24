package services

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// A period sums only its own drives, a drive costs the same whatever period is asked, and the whole history agrees
// with the tag breakdown of the vehicle.
func TestIntegrationDriveSummaryCountsThePeriodAndAgreesWithTheTagBreakdown(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "drive_summary@example.com")

	base := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	for i, cost := range []money.Cents{1200, 1800} {
		c := &models.ChargeLog{VehicleID: v.ID, Date: base.Add(-time.Duration(i+1) * 24 * time.Hour), KwhAdded: 40, Cost: &cost, Currency: "EUR"}
		if err := repo.CreateManualCharge(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	tagsByDrive := [][]string{{"Pro"}, {"Perso"}, {"Pro", "Perso"}, {}}
	for i, tags := range tagsByDrive {
		d := mustDrive(t, repo, v.ID, 100+i, base.Add(time.Duration(i)*24*time.Hour), 20000+float64(i)*100, 30+float64(i)*10)
		if len(tags) > 0 {
			if err := repo.UpdateDriveTags(ctx, d.ID, v.ID, tags); err != nil {
				t.Fatal(err)
			}
		}
	}
	mustDriveWithoutDistance(t, repo, v.ID, 200, base.Add(4*24*time.Hour))

	svc := NewTCOService(db.Pool, "UTC")
	whole, err := svc.DriveSummaryFor(ctx, v.ID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if whole.Drives != 5 || whole.DrivesWithoutDistance != 1 || whole.UntaggedDrives != 2 || whole.DistanceKm != 180 {
		t.Fatalf("whole history: %+v", whole)
	}
	if whole.Cost == 0 || whole.CostPerKm == 0 {
		t.Fatalf("the fixture must price the drives: %+v", whole)
	}

	tco, err := svc.ComputeVehicleTCO(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]TagCostBreakdown{}
	for _, tb := range tco.TagBreakdown {
		want[tb.Tag] = tb
	}
	for _, tb := range whole.ByTag {
		if w := want[tb.Tag]; w.Cost != tb.Cost || w.DistanceKm != tb.DistanceKm || w.SharedDrives != tb.SharedDrives {
			t.Errorf("tag %q: summary %+v, breakdown %+v", tb.Tag, tb, w)
		}
	}

	from, to := base.Add(24*time.Hour), base.Add(2*24*time.Hour)
	period, err := svc.DriveSummaryFor(ctx, v.ID, &from, &to)
	if err != nil {
		t.Fatal(err)
	}
	if period.Drives != 2 || period.DistanceKm != 90 || period.UntaggedDrives != 0 || period.DrivesWithoutDistance != 0 {
		t.Fatalf("period: %+v", period)
	}
	var perso TagCostBreakdown
	for _, tb := range period.ByTag {
		if tb.Tag == "Perso" {
			perso = tb
		}
	}
	if perso.SharedDrives != 1 || perso.DistanceKm != 90 || perso.Cost != want["Perso"].Cost {
		t.Errorf("the two Perso drives of the period are the whole history's: %+v vs %+v", perso, want["Perso"])
	}
	if period.Cost >= whole.Cost {
		t.Errorf("a period costs less than the history: %d vs %d", period.Cost, whole.Cost)
	}
	if empty, err := svc.DriveSummaryFor(ctx, v.ID, &to, &from); err != nil || empty.Drives != 0 || empty.CostPerKm != 0 || len(empty.ByTag) != 0 {
		t.Errorf("an inverted period holds nothing: %+v, %v", empty, err)
	}
}
