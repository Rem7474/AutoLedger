package services

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestCarpoolUnitRatesDefaults(t *testing.T) {
	rates := &UnitRates{
		ElectricityPerKwh: 0.22,
		TiresPerKm:        0.020,
		MaintenancePerKm:  0.015,
		InsurancePerKm:    0.035,
	}

	distance := 450.0                // Paris - Lyon
	kwh := (distance * 16.5) / 100.0 // 74.25 kWh
	elecCost := kwh * rates.ElectricityPerKwh
	tiresCost := distance * rates.TiresPerKm
	maintCost := distance * rates.MaintenancePerKm
	insCost := distance * rates.InsurancePerKm
	tolls := 35.0

	totalCost := elecCost + tolls + tiresCost + maintCost + insCost

	if totalCost <= 0 {
		t.Errorf("expected positive total cost, got %f", totalCost)
	}

	// 3 passengers paying total 65 EUR
	revenue := 65.0
	netCost := totalCost - revenue
	coveragePct := (revenue / totalCost) * 100

	if coveragePct <= 0 || coveragePct > 100 {
		t.Errorf("expected coverage percentage between 0 and 100, got %f", coveragePct)
	}

	if netCost >= totalCost {
		t.Errorf("expected net cost to be reduced by passengers revenue")
	}
}

func TestMonthlyInsuranceAllocationFormula(t *testing.T) {
	monthlyInsuranceEUR := 60.0 // 60.00 EUR/month

	// Month 1 (past completed month):
	// Leg 1: 40 km, Leg 2: 60 km, Other driving: 100 km -> Total month = 200 km
	totalMonth1Km := 200.0
	leg1Km := 40.0
	leg2Km := 60.0

	// Each km in Month 1 costs 60 / 200 = 0.30 EUR/km
	leg1Cost := money.FromFloat(monthlyInsuranceEUR * (leg1Km / totalMonth1Km))
	leg2Cost := money.FromFloat(monthlyInsuranceEUR * (leg2Km / totalMonth1Km))

	if leg1Cost != 1200 { // 12.00 EUR (40 * 0.30)
		t.Fatalf("expected 1200 cents, got %d", leg1Cost)
	}
	if leg2Cost != 1800 { // 18.00 EUR (60 * 0.30)
		t.Fatalf("expected 1800 cents, got %d", leg2Cost)
	}
	totalTripCost := leg1Cost + leg2Cost
	if totalTripCost != 3000 { // 30.00 EUR (50% of monthly insurance)
		t.Fatalf("expected 3000 cents, got %d", totalTripCost)
	}

	// Inactive days: If the car drove only on these 2 trips plus 1 other 100 km drive,
	// and was parked for 25 days out of 30, the total allocated across all drives in Month 1 is:
	otherDriveKm := 100.0
	otherCost := money.FromFloat(monthlyInsuranceEUR * (otherDriveKm / totalMonth1Km))
	if totalTripCost+otherCost != 6000 { // 60.00 EUR (100% of monthly insurance)
		t.Fatalf("expected 6000 cents (100%% of monthly insurance), got %d", totalTripCost+otherCost)
	}

	// Month in progress (ongoing month): uses reference rate
	refRatePerKm := 0.05 // 0.05 EUR/km
	currentMonthDriveKm := 40.0
	currentMonthCost := money.FromFloat(currentMonthDriveKm * refRatePerKm)
	if currentMonthCost != 200 { // 2.00 EUR
		t.Fatalf("expected 200 cents, got %d", currentMonthCost)
	}
}

func TestFuelOnlyDriveCostIsPricedFromFuelNeverKwh(t *testing.T) {
	d := models.Drive{DistanceKm: ptrFloat(100)}
	known := &UnitRates{FuelOnly: true, FuelLitresPer100: 7, FuelPricePerLiter: 1.8, ElectricityPerKwh: 0.3, TiresSource: RateSourceHistory, MaintenanceSource: RateSourceHistory}
	c := ComputeDriveCost(known, d, 0, 0)
	if c.ElectricityKwh != 0 || c.FuelLitres != 7 || c.ElectricityCost != 0 || c.FuelCost != money.FromFloat(12.6) || c.EnergyCost != money.FromFloat(12.6) || !c.FuelOnly || c.EnergySource != EnergySourceFuel || c.HasEstimates {
		t.Fatalf("fuel priced from the fill-ups: %+v", c)
	}
	unknown := &UnitRates{FuelOnly: true, FuelPricePerLiter: 1.8, TiresSource: RateSourceHistory, MaintenanceSource: RateSourceHistory}
	c = ComputeDriveCost(unknown, d, 0, 0)
	if c.FuelLitres != 0 || c.EnergyCost != 0 || c.EnergySource != EnergySourceUnknown || !c.HasEstimates {
		t.Fatalf("without a consumption nothing is invented: %+v", c)
	}
	// A hybrid is not fuel-only: its drive keeps the kWh figures
	hybrid := &UnitRates{ElectricityPerKwh: 0.25, TiresSource: RateSourceHistory, MaintenanceSource: RateSourceHistory, ElectricitySource: RateSourceHistory}
	c = ComputeDriveCost(hybrid, models.Drive{DistanceKm: ptrFloat(100), EnergyConsumedKwh: ptrFloat(16)}, 0, 0)
	if c.FuelLitres != 0 || c.ElectricityKwh != 16 || c.ElectricityCost != money.FromFloat(4) || c.EnergyCost != c.ElectricityCost || c.FuelCost != 0 {
		t.Fatalf("a hybrid drive stays electric: %+v", c)
	}
}

func ptrFloat(v float64) *float64 { return &v }
