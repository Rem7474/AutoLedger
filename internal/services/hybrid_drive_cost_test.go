package services

import (
	"math"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func hybridRates() *UnitRates {
	return &UnitRates{
		Hybrid: true, ElectricityPerKwh: 0.25, ElectricitySource: RateSourceRecentCharges,
		TiresSource: RateSourceHistory, MaintenanceSource: RateSourceHistory,
		FuelLitresPer100: 5, FuelPricePerLiter: 1.5,
	}
}

func hybridDrive(km float64, share, kwh *float64, manual, estimated bool) models.Drive {
	return models.Drive{DistanceKm: &km, ElectricShare: share, EnergyConsumedKwh: kwh, IsManual: manual, EnergyEstimated: estimated}
}

func TestHybridDriveCost(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	tests := []struct {
		name      string
		rates     func() *UnitRates
		drive     models.Drive
		kwh       float64
		litres    float64
		energy    money.Cents
		source    string
		estimates bool
	}{
		{"split: electric kWh plus fuel on the rest", hybridRates, hybridDrive(100, f(0.4), f(8), true, true), 8, 3, money.FromFloat(8*0.25 + 3*1.5), EnergySourceConsumption, false},
		{"typed kWh is measured, the share still gives the fuel", hybridRates, hybridDrive(100, f(0.9), f(5), true, false), 5, 0.5, money.FromFloat(5*0.25 + 0.5*1.5), EnergySourceMeasured, false},
		{"all electric: no fuel part", hybridRates, hybridDrive(100, f(1), f(20), true, true), 20, 0, money.FromFloat(20 * 0.25), EnergySourceConsumption, false},
		{"all on fuel: zero kWh is not replaced by a default", hybridRates, hybridDrive(100, f(0), f(0), true, true), 0, 5, money.FromFloat(5 * 1.5), EnergySourceConsumption, false},
		{"manual drive without share or energy: unknown, empty", hybridRates, hybridDrive(100, nil, nil, true, false), 0, 0, 0, EnergySourceUnknown, true},
		{"manual drive without share but with typed kWh: all electric", hybridRates, hybridDrive(100, nil, f(12), true, false), 12, 0, money.FromFloat(12 * 0.25), EnergySourceMeasured, false},
		{"synchronized drive keeps its measured kWh, no fuel", hybridRates, hybridDrive(100, nil, f(15), false, false), 15, 0, money.FromFloat(15 * 0.25), EnergySourceMeasured, false},
		{"synchronized drive without energy keeps the default estimate", hybridRates, hybridDrive(100, nil, nil, false, false), 16.5, 0, money.FromFloat(16.5 * 0.25), EnergySourceDefault, true},
		{"fuel part without consumption: litres unknown, flagged", func() *UnitRates { r := hybridRates(); r.FuelLitresPer100 = 0; return r }, hybridDrive(100, f(0.4), f(8), true, true), 8, 0, money.FromFloat(8 * 0.25), EnergySourceConsumption, true},
		{"fuel part without price: litres known, cost missing, flagged", func() *UnitRates { r := hybridRates(); r.FuelPricePerLiter = 0; return r }, hybridDrive(100, f(0.4), f(8), true, true), 8, 3, money.FromFloat(8 * 0.25), EnergySourceConsumption, true},
		{"all-electric share needs no fuel rate", func() *UnitRates { r := hybridRates(); r.FuelLitresPer100, r.FuelPricePerLiter = 0, 0; return r }, hybridDrive(100, f(1), f(20), true, true), 20, 0, money.FromFloat(20 * 0.25), EnergySourceConsumption, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeDriveCost(tc.rates(), tc.drive, 0, 0)
			if got.ElectricityKwh != tc.kwh || math.Abs(got.FuelLitres-tc.litres) > 1e-9 || got.EnergyCost != tc.energy || got.EnergySource != tc.source || got.HasEstimates != tc.estimates {
				t.Fatalf("got kWh %v, litres %v, energy %v, source %s, estimates %v", got.ElectricityKwh, got.FuelLitres, got.EnergyCost, got.EnergySource, got.HasEstimates)
			}
			if got.EnergyCost != got.ElectricityCost+got.FuelCost || got.FuelOnly {
				t.Fatalf("energy is the sum of the two parts and a hybrid is not fuel-only: %+v", got)
			}
		})
	}
}
