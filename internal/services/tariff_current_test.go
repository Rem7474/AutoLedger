package services

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func offPeakPlan() *models.TariffPlan {
	peak, off := money.Rate(250000), money.Rate(160000)
	return &models.TariffPlan{
		PlanType: models.TariffTypeTimeOfUse, PeakRateCents: &peak, OffpeakRateCents: &off,
		TimeWindows: []models.TimeWindow{{Start: "22:00", End: "06:00", Kind: models.TimeWindowOffPeak}},
	}
}

func TestPriceAtReadsTheBandOnTheWallClock(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("no tzdata")
	}
	s := &TariffService{loc: paris}
	plan := offPeakPlan()

	// 21:30 UTC in July is 23:30 in Paris: off-peak
	got, ok := s.PriceAt(plan, time.Date(2026, 7, 10, 21, 30, 0, 0, time.UTC))
	if !ok || got.Band != models.TimeWindowOffPeak || got.Rate != 160000 {
		t.Errorf("23:30 local is off-peak: %+v %v", got, ok)
	}
	got, _ = s.PriceAt(plan, time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC))
	if got.Band != models.TimeWindowPeak || got.Rate != 250000 {
		t.Errorf("midday is peak: %+v", got)
	}
	// the window starts and ends on the minute: 22:00 is in, 06:00 is out
	got, _ = s.PriceAt(plan, time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC))
	if got.Band != models.TimeWindowOffPeak {
		t.Errorf("22:00 local belongs to the window: %+v", got)
	}
	got, _ = s.PriceAt(plan, time.Date(2026, 7, 11, 4, 0, 0, 0, time.UTC))
	if got.Band != models.TimeWindowPeak {
		t.Errorf("06:00 local no longer does: %+v", got)
	}
}

func TestPriceAtRefusesWhatCannotBePriced(t *testing.T) {
	s := NewTariffService()
	day := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	if _, ok := s.PriceAt(nil, day); ok {
		t.Error("no plan, no price")
	}
	rate := money.Rate(200000)
	flat := &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &rate, ValidFrom: strPtr("2026-08-01")}
	if _, ok := s.PriceAt(flat, day); ok {
		t.Error("a plan that starts later has no price today")
	}
	flat.ValidFrom = strPtr("2026-07-01")
	if got, ok := s.PriceAt(flat, day); !ok || got.Band != "FLAT" || got.Rate != rate {
		t.Errorf("a flat plan in force prices every minute: %+v %v", got, ok)
	}
	incomplete := offPeakPlan()
	incomplete.OffpeakRateCents = nil
	if _, ok := s.PriceAt(incomplete, day); ok {
		t.Error("a peak / off-peak plan without both prices is unpriced, not priced at zero")
	}
}

func TestNextPriceChangeFindsTheNextBoundary(t *testing.T) {
	s := NewTariffService()
	plan := offPeakPlan()

	at, price, ok := s.NextPriceChange(plan, time.Date(2026, 7, 10, 15, 20, 30, 0, time.UTC))
	if !ok || !at.Equal(time.Date(2026, 7, 10, 22, 0, 0, 0, time.UTC)) || price.Band != models.TimeWindowOffPeak || price.Rate != 160000 {
		t.Errorf("the afternoon ends at 22:00 with the off-peak price: %v %+v %v", at, price, ok)
	}
	at, price, ok = s.NextPriceChange(plan, time.Date(2026, 7, 10, 23, 0, 0, 0, time.UTC))
	if !ok || !at.Equal(time.Date(2026, 7, 11, 6, 0, 0, 0, time.UTC)) || price.Band != models.TimeWindowPeak {
		t.Errorf("the night ends at 06:00 the next day: %v %+v %v", at, price, ok)
	}
	// exactly on the boundary the new band is already in force, so the change is the following one
	at, _, ok = s.NextPriceChange(plan, time.Date(2026, 7, 10, 22, 0, 0, 0, time.UTC))
	if !ok || !at.Equal(time.Date(2026, 7, 11, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("at 22:00 sharp the off-peak price is current: %v %v", at, ok)
	}
}

func TestNextPriceChangeIgnoresBandsThatCostTheSameAndFlatPlans(t *testing.T) {
	s := NewTariffService()
	rate := money.Rate(200000)
	flat := &models.TariffPlan{PlanType: models.TariffTypeFlat, FlatRateCents: &rate}
	if _, _, ok := s.NextPriceChange(flat, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)); ok {
		t.Error("a flat plan never changes")
	}
	same := offPeakPlan()
	same.OffpeakRateCents = same.PeakRateCents
	if _, _, ok := s.NextPriceChange(same, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)); ok {
		t.Error("a window priced like the rest is not a change of price")
	}
}

func TestNextPriceChangeStopsWhereThePlanStopsApplying(t *testing.T) {
	s := NewTariffService()
	plan := offPeakPlan()
	plan.ValidTo = strPtr("2026-07-10")
	if _, _, ok := s.NextPriceChange(plan, time.Date(2026, 7, 10, 23, 0, 0, 0, time.UTC)); ok {
		t.Error("the plan ends at midnight, the following price is unknown rather than guessed")
	}
	at, _, ok := s.NextPriceChange(plan, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	if !ok || !at.Equal(time.Date(2026, 7, 10, 22, 0, 0, 0, time.UTC)) {
		t.Errorf("a change before the end of the plan is still reported: %v %v", at, ok)
	}
}

func TestNextPriceChangeFollowsWeekdayRules(t *testing.T) {
	s := NewTariffService()
	cheap, dear := money.Rate(100000), money.Rate(300000)
	plan := &models.TariffPlan{PlanType: models.TariffTypeBands,
		Bands:       []models.TariffBand{{Name: "WEEKEND", RateCents: cheap}, {Name: "WEEK", RateCents: dear}},
		DefaultBand: "WEEK",
		Rules:       []models.TariffRule{{Days: []int{0, 6}, Start: "00:00", End: "24:00", Band: "WEEKEND"}}}
	// Friday 2026-07-10 noon: the weekend starts on Saturday 00:00
	at, price, ok := s.NextPriceChange(plan, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	if !ok || !at.Equal(time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)) || price.Band != "WEEKEND" {
		t.Errorf("the weekend rate starts on Saturday midnight: %v %+v %v", at, price, ok)
	}
}
