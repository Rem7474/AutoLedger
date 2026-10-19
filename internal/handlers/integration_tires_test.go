package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationTiresReportWearForecastAndLeaveDisposedOnesWithoutADistanceLeft(t *testing.T) {
	bought := time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC)
	mounted := services.TireWearStats{
		Tire:            models.Tire{Brand: "Brand", Season: models.TireSeasonWinter, CurrentPosition: models.TirePosFL, PurchaseDate: bought, PurchasePrice: 52000},
		TotalDistanceKm: 18000, LifeProgressPct: 45.04, CurrentDepthMm: 5.26, MinLegalDepthMm: 1.6, WearPercentage: 41.5, Condition: "GOOD",
		EstimatedRemainingKm: 21999.6, WearRateSource: "measured", LogsCount: 3, CostPerKm: 0.0289,
		ReplacementForecast: &services.TireForecast{ReplacementDate: "2028-01-15"},
	}
	dynamic := mounted
	dynamic.DrivesCount, dynamic.DynamicRemainingKm = 40, 15000
	disposed := services.TireWearStats{Tire: models.Tire{CurrentPosition: models.TirePosDisposed, PurchaseDate: bought}, Condition: "CRITICAL"}

	out := buildIntegrationTires("v1", "CHF", []services.TireWearStats{mounted, dynamic, disposed})

	if len(out.Tires) != 3 {
		t.Fatalf("tires: %+v", out)
	}
	a := out.Tires[0]
	if a.Position != "FL" || a.PurchaseDate != "2025-03-14" || a.PurchasePrice.Amount != "520.00" || a.LifeUsedPct != 45 || a.TreadDepthMm != 5.3 || !a.DepthMeasured ||
		a.RemainingKm == nil || a.RemainingKm.Value != 22000 || a.ReplacementDate == nil || *a.ReplacementDate != "2028-01-15" || a.CostPerKm == nil || a.CostPerKm.Amount != "0.0289" {
		t.Errorf("a mounted tire: %+v", a)
	}
	if out.Tires[1].RemainingKm.Value != 15000 {
		t.Errorf("the distance left follows the driving telemetry when there is some: %+v", out.Tires[1].RemainingKm)
	}
	d := out.Tires[2]
	if d.RemainingKm != nil || d.ReplacementDate != nil || d.CostPerKm != nil || d.DepthMeasured {
		t.Errorf("a disposed tire has nothing left to predict: %+v", d)
	}
	if none := buildIntegrationTires("v1", "CHF", nil); none.Tires == nil {
		t.Error("no tire is an empty list")
	}
}
