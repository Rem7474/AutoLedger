package services

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

// 1000 km a month over the last year, 12 000 km now.
func steadyAnchors(now time.Time) []OdometerAnchor {
	return []OdometerAnchor{
		{Date: now.AddDate(-1, 0, 0), Km: 0 + 1},
		{Date: now, Km: 12000},
	}
}

func TestForecastNeedsMileageHistory(t *testing.T) {
	now := d(2026, 10, 10)
	if got := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 5000, Season: models.TireSeasonSummer}); got != nil {
		t.Fatalf("no anchors must give no forecast, got %+v", got)
	}
	flat := []OdometerAnchor{{Date: now.AddDate(-1, 0, 0), Km: 500}, {Date: now, Km: 500}}
	if got := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 5000, Anchors: flat}); got != nil {
		t.Fatalf("zero mileage must give no forecast, got %+v", got)
	}
	short := []OdometerAnchor{{Date: now.AddDate(0, 0, -5), Km: 100}, {Date: now, Km: 300}}
	if got := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 5000, Anchors: short}); got != nil {
		t.Fatalf("a few days of history must give no forecast, got %+v", got)
	}
}

func TestForecastAllSeasonRunsAllYear(t *testing.T) {
	now := d(2026, 10, 10)
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 6000, Season: models.TireSeasonAllSeason, Anchors: steadyAnchors(now)})
	if f == nil || f.MonthsSource != TireForecastMonthsDefault || len(f.MountedMonths) != 12 {
		t.Fatalf("unexpected forecast %+v", f)
	}
	// ~1000 km/month: six months later, early April.
	date, _ := time.Parse("2006-01-02", f.ReplacementDate)
	if date.Before(d(2027, 3, 25)) || date.After(d(2027, 4, 20)) {
		t.Fatalf("replacement date %s out of range", f.ReplacementDate)
	}
}

func TestForecastSkipsMonthsOffTheCar(t *testing.T) {
	now := d(2026, 10, 10)
	in := TireForecastInput{Now: now, RemainingKm: 6000, Season: models.TireSeasonSummer, Anchors: steadyAnchors(now)}
	f := ForecastTireReplacement(in)
	if f == nil {
		t.Fatal("expected a forecast")
	}
	// Summer tires: the rest of October (~0.7 month), then storage until April; the remaining ~5300 km are driven from April on.
	date, _ := time.Parse("2006-01-02", f.ReplacementDate)
	if date.Before(d(2027, 8, 20)) || date.After(d(2027, 9, 25)) {
		t.Fatalf("summer tires should last into late summer 2027, got %s", f.ReplacementDate)
	}

	in.Season = models.TireSeasonWinter
	w := ForecastTireReplacement(in)
	wd, _ := time.Parse("2006-01-02", w.ReplacementDate)
	// Winter tires in October: storage until November, then 5 months a year.
	if wd.Before(d(2027, 12, 1)) {
		t.Fatalf("winter tires should wear out over the following winters, got %s", w.ReplacementDate)
	}
}

func TestForecastLearnsMonthsFromFinishedSessions(t *testing.T) {
	now := d(2026, 10, 10)
	end := d(2026, 3, 15)
	sessions := []models.TireMountSession{
		{MountedDate: d(2025, 12, 1), DismountedDate: &end},
		{MountedDate: d(2026, 9, 1)}, // still mounted: says nothing about the end of the season
	}
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 3000, Season: models.TireSeasonWinter, SeasonSessions: sessions, Anchors: steadyAnchors(now)})
	if f == nil || f.MonthsSource != TireForecastMonthsLearned {
		t.Fatalf("expected learned months, got %+v", f)
	}
	want := []int{1, 2, 3, 12}
	if len(f.MountedMonths) != len(want) {
		t.Fatalf("mounted months %v, want %v", f.MountedMonths, want)
	}
	for i, m := range want {
		if f.MountedMonths[i] != m {
			t.Fatalf("mounted months %v, want %v", f.MountedMonths, want)
		}
	}
}

func TestForecastWornTireIsDueNow(t *testing.T) {
	now := d(2026, 10, 10)
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 0, Season: models.TireSeasonSummer, Anchors: steadyAnchors(now)})
	if f == nil || f.ReplacementDate != "2026-10-10" {
		t.Fatalf("a worn tire is due today, got %+v", f)
	}
}

func TestForecastBeyondHorizonHasNoDate(t *testing.T) {
	now := d(2026, 10, 10)
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 1_000_000, Season: models.TireSeasonAllSeason, Anchors: steadyAnchors(now)})
	if f == nil || f.ReplacementDate != "" {
		t.Fatalf("expected a forecast without date, got %+v", f)
	}
}
