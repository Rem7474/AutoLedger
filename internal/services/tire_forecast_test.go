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
		{MountedDate: d(2026, 9, 1)}, // still mounted: its months so far are on the car too
	}
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 3000, Season: models.TireSeasonWinter, SeasonSessions: sessions, Anchors: steadyAnchors(now)})
	if f == nil || f.MonthsSource != TireForecastMonthsLearned {
		t.Fatalf("expected learned months, got %+v", f)
	}
	want := []int{1, 2, 3, 9, 10, 12}
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

// Summer months at 2000 km, winter months at 500 km: a winter tire must be forecast with the winter mileage.
func TestForecastUsesTheMileageOfTheMountedMonths(t *testing.T) {
	now := d(2026, 10, 10)
	var anchors []OdometerAnchor
	km := 0.0
	for m := d(2024, 10, 1); m.Before(now); m = m.AddDate(0, 1, 0) {
		anchors = append(anchors, OdometerAnchor{Date: m, Km: km + 1})
		if mo := m.Month(); mo >= 4 && mo <= 10 {
			km += 2000
		} else {
			km += 500
		}
	}
	anchors = append(anchors, OdometerAnchor{Date: now, Km: km + 1})

	winter := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 1250, Season: models.TireSeasonWinter, Anchors: anchors})
	summer := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 1250, Season: models.TireSeasonSummer, Anchors: anchors})
	if winter == nil || summer == nil {
		t.Fatal("expected forecasts")
	}
	if winter.MonthlyKm < 450 || winter.MonthlyKm > 550 {
		t.Fatalf("winter monthly km %.0f, want ~500", winter.MonthlyKm)
	}
	if summer.MonthlyKm < 1900 || summer.MonthlyKm > 2100 {
		t.Fatalf("summer monthly km %.0f, want ~2000", summer.MonthlyKm)
	}
	// 1250 km at 500 km/month in winter: Nov 1 + 2.5 months, mid-January.
	wd, _ := time.Parse("2006-01-02", winter.ReplacementDate)
	if wd.Before(d(2027, 1, 5)) || wd.After(d(2027, 1, 25)) {
		t.Fatalf("winter replacement %s, want mid-January 2027", winter.ReplacementDate)
	}
}

// Summer tires kept on all year (no winter tires required): the open session that already spans the winter says so.
func TestForecastRecognisesASetKeptOnAllYear(t *testing.T) {
	now := d(2026, 12, 15)
	open := []models.TireMountSession{{MountedDate: d(2026, 4, 1)}}
	f := ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 3000, Season: models.TireSeasonSummer, SeasonSessions: open, Anchors: steadyAnchors(now)})
	if f == nil || f.MonthsSource != TireForecastMonthsLearned {
		t.Fatalf("expected learned months, got %+v", f)
	}
	if got := len(f.MountedMonths); got != 9 { // April to October by default, plus November and December already driven
		t.Fatalf("mounted months %v", f.MountedMonths)
	}

	// A full year on the car covers every month.
	long := []models.TireMountSession{{MountedDate: d(2025, 4, 1)}}
	f = ForecastTireReplacement(TireForecastInput{Now: now, RemainingKm: 3000, Season: models.TireSeasonSummer, SeasonSessions: long, Anchors: steadyAnchors(now)})
	if len(f.MountedMonths) != 12 {
		t.Fatalf("a set mounted for over a year runs all year, got %v", f.MountedMonths)
	}
	date, _ := time.Parse("2006-01-02", f.ReplacementDate)
	if date.After(d(2027, 4, 1)) { // ~1000 km/month, no storage
		t.Fatalf("year-round set should be due by spring, got %s", f.ReplacementDate)
	}
}

func TestKeptOnAllYear(t *testing.T) {
	sum, win, all := models.TireSeasonSummer, models.TireSeasonWinter, models.TireSeasonAllSeason
	cases := []struct {
		name    string
		season  models.TireSeason
		present []models.TireSeason
		want    bool
	}{
		{"summer alone", sum, []models.TireSeason{sum, sum}, true},
		{"winter alone", win, []models.TireSeason{win}, true},
		{"summer with a winter set", sum, []models.TireSeason{sum, win}, false},
		{"winter with a summer set", win, []models.TireSeason{win, sum}, false},
		{"summer with all-season tires", sum, []models.TireSeason{sum, all}, false},
		{"all-season", all, []models.TireSeason{all}, false},
	}
	for _, c := range cases {
		if got := KeptOnAllYear(c.season, c.present); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestForecastRunsAllYearWithoutAnotherSet(t *testing.T) {
	now := d(2026, 10, 10)
	in := TireForecastInput{Now: now, RemainingKm: 3000, Season: models.TireSeasonSummer, Anchors: steadyAnchors(now), KeptAllYear: true}
	f := ForecastTireReplacement(in)
	if f == nil || f.MonthsSource != TireForecastMonthsAllYear || len(f.MountedMonths) != 12 {
		t.Fatalf("expected twelve months, got %+v", f)
	}
	// Finished sessions still win over the deduction.
	end := d(2026, 3, 15)
	in.SeasonSessions = []models.TireMountSession{{MountedDate: d(2025, 4, 1), DismountedDate: &end}}
	if f = ForecastTireReplacement(in); f.MonthsSource != TireForecastMonthsLearned {
		t.Fatalf("history must take precedence, got %+v", f)
	}
}
