package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationEnergyReadsAnElectricVehiclesPricesAndBatteryOverAYear(t *testing.T) {
	v := &models.Vehicle{ID: "v1", Currency: "CHF", Powertrain: models.PowertrainEV}
	stats := &services.EnergyStats{
		Summary: services.EnergySummary{ConsumptionKwh100km: f64(17.5), PricePerKwh: f64(0.215), CostPer100km: f64(3.76), ChargeEfficiency: f64(0.912),
			CostPerFullCharge: f64(14.4), SessionsWithoutCost: 2, EstimatedCapacityKwh: f64(72.4)},
		ChargeClasses: []services.ChargeClassStat{
			{Class: services.ChargeClassAC, Sessions: 12, KwhAdded: 400, EnergyCost: money.Cents(8600), PricePerKwh: f64(0.215), ChargeEfficiency: f64(0.9)},
			{Class: services.ChargeClassDC, Sessions: 3, KwhAdded: 150, EnergyCost: money.Cents(7500), SessionsWithoutCost: 1},
		},
		Temperature: services.TemperatureEffect{ColdConsumptionKwh100km: f64(21), MildConsumptionKwh100km: f64(16), ExtraPercent: f64(31.3), ExtraCostPer100km: f64(1.07)},
		BatteryHealth: []models.BatterySnapshot{
			{Date: "2025-10-15", HealthPercent: f64(97.2)},
			{Date: "2026-03-01", HealthPercent: f64(95.9)},
			{Date: "2026-10-12", HealthPercent: f64(94.1), CurrentCapacityKwh: f64(70.5), MaxCapacityKwh: f64(75)},
		},
		Basis: services.EnergyBasis{Consumption: services.BasisMeasured, Cost: services.BasisMeasured, Battery: services.BasisMeasured},
	}
	out := buildIntegrationEnergy(v, stats, nil)

	e := out.Electric
	if out.Fuel != nil || e == nil || e.Kwh100km == nil || *e.Kwh100km != 17.5 || e.PricePerKwh.Amount != "0.215" || e.CostPer100km.Amount != "3.76" ||
		e.CostPer100km.Currency != "CHF" || *e.Efficiency != 91.2 || e.FullCharge.Amount != "14.40" || *e.Capacity != 72.4 || e.WithoutCost != 2 {
		t.Fatalf("electric summary: %+v", e)
	}
	if len(e.ChargeMix) != 2 || e.ChargeMix[0].Class != "ac" || e.ChargeMix[0].EnergyCost.Amount != "86.00" || *e.ChargeMix[0].ChargeEfficiency != 90 ||
		e.ChargeMix[1].PricePerKwh != nil || e.ChargeMix[1].WithoutCost != 1 {
		t.Errorf("charge mix: %+v", e.ChargeMix)
	}
	if e.Temperature == nil || *e.Temperature.ExtraPercent != 31.3 || e.Temperature.ExtraCost.Amount != "1.07" {
		t.Errorf("temperature: %+v", e.Temperature)
	}
	b := e.BatteryHealth
	if b == nil || b.LatestPercent != 94.1 || b.LatestDate != "2026-10-12" || b.ChangePoints12m == nil || *b.ChangePoints12m != -3.1 || b.ReadingsRecorded != 3 || *b.CapacityKwh != 70.5 {
		t.Errorf("battery health, a year apart: %+v", b)
	}
	if e.Basis["consumption"] != "measured" || e.Basis["charging"] != "" {
		t.Errorf("basis: %+v", e.Basis)
	}
}

func TestIntegrationBatteryHealthDoesNotInventAYearOfHistory(t *testing.T) {
	short := buildIntegrationBattery([]models.BatterySnapshot{{Date: "2026-08-01", HealthPercent: f64(96)}, {Date: "2026-10-12", HealthPercent: f64(95)}})
	if short == nil || short.ChangePoints12m != nil || short.LatestPercent != 95 {
		t.Errorf("two months of history give no yearly change: %+v", short)
	}
	if buildIntegrationBattery(nil) != nil || buildIntegrationBattery([]models.BatterySnapshot{{Date: "2026-08-01"}}) != nil {
		t.Error("no health reading means no battery section")
	}
}

func TestIntegrationEnergyOfAFuelVehicleHasNoElectricSide(t *testing.T) {
	v := &models.Vehicle{ID: "v2", Currency: "EUR", Powertrain: models.PowertrainICE}
	sum := &services.TCOSummary{FuelFillUps: 20, TotalLiters: 900, AvgCostPerLiter: 1.789, ConsumptionL100km: f64(6.4),
		EnergyCost: money.Cents(160000), FuelEnergyCost: money.Cents(160000), EnergyCostPerKm: f64(0.115)}
	out := buildIntegrationEnergy(v, nil, sum)

	f := out.Fuel
	if out.Electric != nil || f == nil || f.FillUps != 20 || f.PricePerLitre.Amount != "1.789" || *f.LitresPer100 != 6.4 || f.CostPer100km == nil || f.CostPer100km.Amount != "11.50" {
		t.Errorf("fuel: %+v", f)
	}
	none := buildIntegrationEnergy(v, nil, &services.TCOSummary{})
	if none.Fuel.PricePerLitre != nil || none.Fuel.CostPer100km != nil {
		t.Errorf("without fill-ups nothing is invented: %+v", none.Fuel)
	}
}
