package services

import (
	"math"
	"testing"
	"time"
)

// The period of a priced session comes from the caller, so the number of steps walked must not follow it.
func TestSessionSamplingBoundsTheWalkWhateverThePeriod(t *testing.T) {
	const maxSteps = maxSampledSessionMinutes/5 + 1
	for _, total := range []int{1, 60, 1440, 1441, 7 * 1440, 366 * 1440, 366*1440 + 1, 50 * 366 * 1440, 150_000_000, math.MaxInt32} {
		step, sampled := sessionSampling(total)
		if steps := (sampled + step - 1) / step; steps > maxSteps {
			t.Errorf("%d minutes: %d steps, more than %d", total, steps, maxSteps)
		}
		if total <= maxSampledSessionMinutes && sampled != total {
			t.Errorf("%d minutes (up to a year) must be walked whole, sampled %d", total, sampled)
		}
	}
	if step, _ := sessionSampling(1440); step != 1 {
		t.Errorf("a day is walked minute by minute, got a step of %d", step)
	}
	if step, _ := sessionSampling(1441); step != 5 {
		t.Errorf("beyond a day the step is 5 minutes, got %d", step)
	}
}

// A period of centuries is still priced, from a sample of its first year, and the shares add up to the energy.
func TestPriceSessionOfAHugePeriodStillSplitsTheEnergy(t *testing.T) {
	svc := NewTariffServiceIn("UTC")
	start := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)
	pricing, err := svc.PriceSession(bandsPlan(), start, start.Add(290*365*24*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pricing.Bands) == 0 || pricing.Cost <= 0 {
		t.Fatalf("expected a priced split, got %+v", pricing)
	}
	var kwh float64
	for _, b := range pricing.Bands {
		kwh += b.Kwh
	}
	if kwh < 9.999 || kwh > 10.001 {
		t.Fatalf("the shares must add up to the energy of the session, got %v kWh", kwh)
	}
}

// Usual sessions keep their prices.
func TestPriceSessionKeepsTheSamplingOfUsualPeriods(t *testing.T) {
	svc := NewTariffServiceIn("UTC")
	plan := bandsPlan()
	start := time.Date(2026, 10, 7, 19, 0, 0, 0, time.UTC)
	got, err := svc.CalculateSessionCost(plan, start, start.Add(2*time.Hour), 10)
	if err != nil || got != 350 {
		t.Fatalf("half red, half white: want 350, got %d (err %v)", got, err)
	}
	week, err := svc.PriceSession(plan, start, start.Add(7*24*time.Hour), 70)
	if err != nil || len(week.Bands) != 3 {
		t.Fatalf("a week crosses the three bands, got %+v (err %v)", week, err)
	}
}
