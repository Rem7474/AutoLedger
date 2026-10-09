package services

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

func TestEnergyStatisticsDistinguishFreeAndUnknownCost(t *testing.T) {
	zero := money.Cents(0)
	start := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	from, to := 20, 80
	for _, known := range []bool{true, false} {
		charge := energyCharge{Month: "2026-10", Start: start, End: &end, KwhAdded: 20, StartSoc: &from, EndSoc: &to}
		if known {
			charge.Cost = &zero
		}
		stats := computeEnergyStats([]energyDriveMonth{{Month: "2026-10", DistanceKm: 300, MeasuredKm: 300, Kwh: 20}}, []energyCharge{charge}, nil)
		metrics := []*float64{stats.Summary.PricePerKwh, stats.Summary.CostPer100km, stats.Summary.CostPerFullCharge, stats.Months[0].PricePerKwh, stats.Months[0].CostPer100km, stats.Months[0].CostPer100kmTrailing, stats.ChargeClasses[0].PricePerKwh, stats.ChargeClasses[0].CostPerFullCharge}
		for _, metric := range metrics {
			if known && (metric == nil || *metric != 0) {
				t.Fatalf("known free charge must yield zero: %+v", stats)
			}
			if !known && metric != nil {
				t.Fatalf("unknown charge must remain unavailable: %+v", stats)
			}
		}
		if known && stats.Basis.Cost == "unavailable" {
			t.Fatal("known zero cost is available")
		}
	}
}

func TestEnergyFreeAndPaidPricesAreWeighted(t *testing.T) {
	free, paid := money.Cents(0), money.Cents(1000)
	stats := computeEnergyStats(nil, []energyCharge{{Month: "2026-10", KwhAdded: 20, Cost: &free}, {Month: "2026-10", KwhAdded: 20, Cost: &paid}}, nil)
	if stats.Summary.PricePerKwh == nil || *stats.Summary.PricePerKwh != 0.25 {
		t.Fatalf("free energy must contribute to priced energy: %+v", stats.Summary)
	}
}

func TestCostRatiosNeedAValidDenominator(t *testing.T) {
	for _, pair := range [][2]float64{{0, 0}, {-1, 20}, {1, 0}} {
		if costRatioPtr(pair[0], pair[1], 2) != nil {
			t.Fatalf("invalid cost ratio: %v", pair)
		}
	}
}
