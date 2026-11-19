package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationDriveSummaryReportsTheGapsBesideTheFigures(t *testing.T) {
	out := buildIntegrationDriveSummary("v1", "CHF", false, "2026-08-01", "2026-08-31", services.DriveSummary{
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

	none := buildIntegrationDriveSummary("v1", "CHF", false, "", "", services.DriveSummary{})
	if none.CostPerKm != nil || none.Tags == nil || len(none.Tags) != 0 {
		t.Errorf("no drive: no rate, an empty list: %+v", none)
	}
}

func TestIntegrationDriveSummaryEnergyFollowsWhatTheVehicleBurns(t *testing.T) {
	sum := services.DriveSummary{Drives: 2, DistanceKm: 100, EnergyKwh: 0, FuelLiters: 7.5,
		ByTag: []services.TagCostBreakdown{{Tag: "pro", DistanceKm: 100, FuelOnly: true, FuelLiters: 7.5}}}

	fuel := buildIntegrationDriveSummary("v1", "EUR", true, "", "", sum)
	if fuel.EnergyKwh != nil || fuel.FuelLiters == nil || *fuel.FuelLiters != 7.5 {
		t.Errorf("a fuel vehicle has litres and no kWh, not a 0: kWh %v litres %v", fuel.EnergyKwh, fuel.FuelLiters)
	}
	if fuel.Tags[0].EnergyKwh != nil || fuel.Tags[0].FuelLiters == nil || *fuel.Tags[0].FuelLiters != 7.5 {
		t.Errorf("the tag rows follow: %+v", fuel.Tags[0])
	}

	unknown := buildIntegrationDriveSummary("v1", "EUR", true, "", "", services.DriveSummary{Drives: 1, DistanceKm: 10})
	if unknown.EnergyKwh != nil || unknown.FuelLiters != nil {
		t.Errorf("litres not known yet stay null: %+v", unknown)
	}

	electric := buildIntegrationDriveSummary("v1", "EUR", false, "", "", services.DriveSummary{Drives: 1, DistanceKm: 10, EnergyKwh: 3})
	if electric.EnergyKwh == nil || *electric.EnergyKwh != 3 || electric.FuelLiters != nil {
		t.Errorf("an electric vehicle keeps its kWh and has no litres: %+v", electric)
	}
	zero := buildIntegrationDriveSummary("v1", "EUR", false, "", "", services.DriveSummary{})
	if zero.EnergyKwh == nil || *zero.EnergyKwh != 0 {
		t.Errorf("an electric vehicle with no energy measured reports 0 kWh: %+v", zero)
	}
}
