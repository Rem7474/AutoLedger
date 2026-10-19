package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func fleetFixture() *models.FleetSummaryResponse {
	budget, annual := money.Cents(60000), money.Cents(480000)
	chf := "CHF"
	perKm := func(v float64) *float64 { return &v }
	return &models.FleetSummaryResponse{
		TotalVehicles: 3, Currency: "CHF", CurrentMonth: "2026-10", MonthlyBudget: &budget, BudgetCurrency: &chf,
		Totals: []models.FleetCurrencyTotal{{Currency: "CHF", VehicleCount: 2, CurrentMonthCost: 45000}, {Currency: "EUR", VehicleCount: 1, CurrentMonthCost: 9000}},
		Vehicles: []models.VehicleFleetMetric{
			{VehicleID: "a", Name: "Daily", Currency: "CHF", Powertrain: "EV", MonthCost: 30000, FullCostPerKm: perKm(0.61), RunningCostPerKm: perKm(0.2), Comparable: true, AnnualCost: &annual, CompletenessPct: 90},
			{VehicleID: "b", Name: "Van", Currency: "CHF", Powertrain: "ICE", MonthCost: 15000, FullCostPerKm: perKm(0.48), Comparable: true, CompletenessPct: 85},
			{VehicleID: "c", Name: "Old", Currency: "EUR", Powertrain: "ICE", MonthCost: 9000, FullCostPerKm: perKm(0.1), Comparable: false, CompletenessPct: 40},
		},
		MonthlyCosts:   []models.FleetMonthlyCost{{Month: "2026-09", TotalCost: 50000, EnergyCost: 10000, OtherCost: 40000, DistanceKm: 900}, {Month: "2026-10", TotalCost: 45000, EnergyCost: 8000, OtherCost: 37000, DistanceKm: 700}},
		MemberKmShares: []models.MemberKmShare{{PersonID: "p1", DisplayName: "Alex", DistanceKm: 600, Percentage: 75}, {PersonID: "p2", DisplayName: "Sam", DistanceKm: 200, Percentage: 25}},
	}
}

func TestIntegrationFleetRanksOnlyComparableVehiclesWithinTheirCurrency(t *testing.T) {
	out := buildIntegrationFleet(fleetFixture())

	if out.VehicleCount != 3 || len(out.Vehicles) != 3 || len(out.Totals) != 2 || out.Totals[1].CurrentMonthCost != (IntegrationAmount{Amount: "90.00", Currency: "EUR"}) {
		t.Fatalf("totals stay per currency: %+v", out.Totals)
	}
	van, daily, old := out.Vehicles[1], out.Vehicles[0], out.Vehicles[2]
	if van.CostRank == nil || *van.CostRank != 1 || daily.CostRank == nil || *daily.CostRank != 2 || old.CostRank != nil || old.Comparable {
		t.Errorf("the cheaper comparable vehicle ranks first, the incomparable one is not ranked: %+v %+v %+v", van.CostRank, daily.CostRank, old.CostRank)
	}
	if daily.FullCostPerKm.Amount != "0.6100" || daily.AnnualCost == nil || daily.AnnualCost.Amount != "4800.00" || van.AnnualCost != nil {
		t.Errorf("rates and annual cost: %+v", daily)
	}
}

func TestIntegrationFleetBudgetUsesTheVehiclesOfItsCurrencyAndDriversAreNamed(t *testing.T) {
	res := fleetFixture()
	out := buildIntegrationFleet(res)

	b := out.Budget
	if b == nil || b.Spent.Amount != "450.00" || b.Remaining.Amount != "150.00" || b.UsedPct != 75 || b.Monthly.Currency != "CHF" {
		t.Errorf("the budget counts the CHF vehicles only: %+v", b)
	}
	if len(out.Drivers) != 2 || out.Drivers[0].Name != "Alex" || out.Drivers[0].Distance.Value != 600 {
		t.Errorf("drivers are named: %+v", out.Drivers)
	}
	if len(out.MonthlyCosts) != 2 || out.MonthlyCosts[1].Total.Amount != "450.00" || out.FleetCurrency != "CHF" {
		t.Errorf("monthly series: %+v", out.MonthlyCosts)
	}

	eur := "EUR"
	res.BudgetCurrency = &eur
	res.Totals = res.Totals[:1]
	if buildIntegrationFleet(res).Budget != nil {
		t.Error("a budget in a currency no vehicle uses is not reported against another currency's spending")
	}
	res.MonthlyBudget = nil
	if buildIntegrationFleet(res).Budget != nil {
		t.Error("no budget set")
	}
}
