package services

import (
	"testing"

	"github.com/teslacost/teslacost/internal/money"
)

func TestMaintenanceAmortizationIncludesMonthsWithoutMileage(t *testing.T) {
	for _, mode := range []string{"DURATION", "HYBRID"} {
		t.Run(mode, func(t *testing.T) {
			item := &MaintenanceAmortItem{ID: "service", Month: "2025-01", AmountEur: 120000, Mode: mode, CoverageMonths: 12, CoverageKm: 100000}
			result := calculateMaintenanceAmortization([]*MaintenanceAmortItem{item}, map[string]float64{"2025-01": 100, "2026-10": 100}, "2026-10")
			var total money.Cents
			for _, amount := range result {
				total += amount
			}
			if total != item.AmountEur || result["2025-02"] != 10000 || result["2025-12"] != 10000 || result["2026-10"] != 0 {
				t.Fatalf("calendar coverage: %v, total %v", result, total)
			}
		})
	}
}

func TestMaintenanceGapClosureAndDistanceCoverage(t *testing.T) {
	closing := "service"
	items := []*MaintenanceAmortItem{
		{ID: closing, Month: "2025-01", AmountEur: 120000, Mode: "DURATION", CoverageMonths: 12},
		{ID: "replacement", Month: "2025-04", Mode: "NONE", ClosesMaintenanceID: &closing},
	}
	result := calculateMaintenanceAmortization(items, nil, "2025-10")
	if result["2025-01"]+result["2025-02"]+result["2025-03"] != 30000 || result["2025-04"] != 0 {
		t.Fatalf("closure must stop coverage before April: %v", result)
	}
	items[0].Mode, items[0].CoverageKm = "DISTANCE", 1000
	items = items[:1]
	result = calculateMaintenanceAmortization(items, map[string]float64{"2025-01": 100, "2025-10": 100}, "2025-10")
	if result["2025-02"] != 0 || result["2025-01"]+result["2025-10"] != 24000 {
		t.Fatalf("distance coverage must not advance in parked months: %v", result)
	}
}
