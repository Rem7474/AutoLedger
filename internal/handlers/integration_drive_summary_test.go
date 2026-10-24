package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationDriveSummaryReportsTheGapsBesideTheFigures(t *testing.T) {
	out := buildIntegrationDriveSummary("v1", "CHF", "2026-08-01", "2026-08-31", services.DriveSummary{
		Drives: 5, DrivesWithoutDistance: 1, DistanceKm: 140, EnergyKwh: 25, TollsAmount: 350, Cost: 4900, CostPerKm: 0.35,
		UntaggedDrives: 2, ToQualify: 3, TripGroups: 1,
		ByTag: []services.TagCostBreakdown{{Tag: "pro", DistanceKm: 100, Cost: 3500, CostPerKm: 0.35}},
	})
	if out.Drives != 5 || out.DrivesWithoutDistance != 1 || out.UntaggedDrives != 2 || out.DrivesToQualify != 3 || out.TripGroups != 1 {
		t.Errorf("counts: %+v", out)
	}
	if out.Cost.Amount != "49.00" || out.Tolls.Amount != "3.50" || out.CostPerKm == nil || out.CostPerKm.Amount != "0.3500" || out.Currency != "CHF" {
		t.Errorf("amounts: %+v", out)
	}
	if len(out.Tags) != 1 || out.Tags[0].Tag == nil || *out.Tags[0].Tag != "pro" || out.From != "2026-08-01" {
		t.Errorf("tags and period: %+v", out)
	}

	none := buildIntegrationDriveSummary("v1", "CHF", "", "", services.DriveSummary{})
	if none.CostPerKm != nil || none.Tags == nil || len(none.Tags) != 0 {
		t.Errorf("no drive: no rate, an empty list: %+v", none)
	}
}
