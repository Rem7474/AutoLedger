package services

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestApplicableTariffDistinguishesFreeFromUnknown(t *testing.T) {
	zero, tiny, paid := money.Rate(0), money.Rate(100), money.Rate(250000)
	endDate := "2026-09-30"
	start := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		plan  *models.TariffPlan
		known bool
		cost  money.Cents
	}{
		{"absent", nil, false, 0},
		{"no rate", &models.TariffPlan{PlanType: models.TariffTypeFlat}, false, 0},
		{"expired", &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &zero, ValidTo: &endDate}, false, 0},
		{"flat free", &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &zero}, true, 0},
		{"free timed", &models.TariffPlan{PlanType: models.TariffTypeTimeOfUse, PeakRateCents: &zero, OffpeakRateCents: &zero, TimeWindows: []models.TimeWindow{{Start: "00:00", End: "08:00", Kind: models.TimeWindowOffPeak}}}, true, 0},
		{"incomplete timed", &models.TariffPlan{PlanType: models.TariffTypeTimeOfUse, TimeWindows: []models.TimeWindow{{Start: "00:00", End: "08:00", Kind: models.TimeWindowOffPeak}}}, false, 0},
		{"rounded free", &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &tiny}, true, 0},
		{"free bands", &models.TariffPlan{PlanType: models.TariffTypeBands, Bands: []models.TariffBand{{Name: "Free", RateCents: zero}}, DefaultBand: "Free"}, true, 0},
		{"paid", &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &paid}, true, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewTariffService().CalculateSessionCostIfApplicable(tc.plan, start, start.Add(time.Hour), 20)
			if err != nil || (got != nil) != tc.known {
				t.Fatalf("got=%v, err=%v", got, err)
			}
			if got != nil && *got != tc.cost {
				t.Fatalf("cost=%v, want %v", *got, tc.cost)
			}
		})
	}
}
