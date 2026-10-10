package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type fakeTireWearStore struct {
	logs        []models.TireLog
	sessions    []models.TireMountSession
	logsErr     error
	sessionsErr error
	powerMax    float64
	powerMin    float64
	consumption float64
	drives      int
	telemetry   error
	gotRanges   []database.OdometerRange
	calls       int
}

func (f *fakeTireWearStore) ListTireLogs(context.Context, string) ([]models.TireLog, error) {
	return f.logs, f.logsErr
}

func (f *fakeTireWearStore) ListTireMountSessions(context.Context, string) ([]models.TireMountSession, error) {
	return f.sessions, f.sessionsErr
}

func (f *fakeTireWearStore) GetDrivingTelemetryStats(_ context.Context, _ string, ranges []database.OdometerRange) (float64, float64, float64, int, error) {
	f.calls++
	f.gotRanges = ranges
	return f.powerMax, f.powerMin, f.consumption, f.drives, f.telemetry
}

func wearTire(pos models.TirePosition) *models.Tire {
	vid := "veh-1"
	return &models.Tire{ID: "t1", VehicleID: &vid, CurrentPosition: pos}
}

func TestCalculateTireWearStoreErrors(t *testing.T) {
	boom := errors.New("boom")
	for name, store := range map[string]*fakeTireWearStore{
		"logs":     {logsErr: boom},
		"sessions": {sessionsErr: boom},
	} {
		if _, err := NewTireWearService(store).CalculateTireWear(context.Background(), wearTire(models.TirePosFL), 0); !errors.Is(err, boom) {
			t.Errorf("%s: expected store error, got %v", name, err)
		}
	}
}

func TestCalculateTireWearDefaultsWithoutLogs(t *testing.T) {
	store := &fakeTireWearStore{}
	tire := wearTire(models.TirePosStorage)
	got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if got.InitialDepthMm != 8 || got.MinLegalDepthMm != 1.6 || got.EstimatedLifespanKm != 40000 {
		t.Errorf("defaults not applied: %+v", got)
	}
	if got.CurrentDepthMm != 8 || got.WearPercentage != 0 || got.Condition != "GOOD" {
		t.Errorf("a new tire without logs should be unworn: %+v", got)
	}
	if got.WearRatePer10kKm != 1.2 {
		t.Errorf("expected fallback wear rate 1.2, got %v", got.WearRatePer10kKm)
	}
	if store.calls != 0 || got.DrivingStyle != "" || got.WearExplanation != nil {
		t.Errorf("a stored tire without sessions must not query telemetry")
	}
}

func TestCalculateTireWearMeasuredRate(t *testing.T) {
	// Two logs, 10 000 km apart on a tire mounted since odometer 0: 2 mm worn per 10 000 km.
	store := &fakeTireWearStore{
		logs: []models.TireLog{
			{DepthMm: 6, Odometer: 10000},
			{DepthMm: 8, Odometer: 0},
		},
		sessions: []models.TireMountSession{{MountedOdometer: 0}},
	}
	tire := wearTire(models.TirePosStorage)
	got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
	if err != nil {
		t.Fatal(err)
	}
	// 2 mm worn weighs 2/3 against the 1.2 mm average rate: 2/3*2 + 1/3*1.2.
	if got.WearRatePer10kKm != 1.73 || got.WearRateConfidence != 0.67 {
		t.Errorf("expected blended rate 1.73 at confidence 0.67, got %v at %v", got.WearRatePer10kKm, got.WearRateConfidence)
	}
	// (6 - 1.6) mm remaining at the blended rate.
	if math.Abs(got.EstimatedRemainingKm-4.4/(2.0/3*2+1.2/3)*10000) > 1 {
		t.Errorf("unexpected remaining km, got %v", got.EstimatedRemainingKm)
	}
	if got.DistanceTraveledKm != 10000 || got.LogsCount != 2 {
		t.Errorf("unexpected distance/logs: %+v", got)
	}
}

func TestCalculateTireWearSingleLogCountsInitialDistance(t *testing.T) {
	store := &fakeTireWearStore{
		logs:     []models.TireLog{{DepthMm: 7, Odometer: 5000}},
		sessions: []models.TireMountSession{{MountedOdometer: 0}},
	}
	tire := wearTire(models.TirePosStorage)
	tire.InitialDistanceKm = 5000
	got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 5000)
	if err != nil {
		t.Fatal(err)
	}
	// 1 mm worn over 5000 (before) + 5000 (mount session) km.
	if got.DistanceTraveledKm != 10000 {
		t.Errorf("expected 10000 km, got %v", got.DistanceTraveledKm)
	}
	// 1 mm worn weighs the same as the average rate: (1 + 1.2) / 2.
	if got.WearRatePer10kKm != 1.1 {
		t.Errorf("expected 1.1 mm/10k km, got %v", got.WearRatePer10kKm)
	}
}

func TestCalculateTireWearShortOrTinyWearUsesFallback(t *testing.T) {
	// Distance below 500 km and wear below 0.05 mm are too noisy to extrapolate.
	for name, logs := range map[string][]models.TireLog{
		"short distance": {{DepthMm: 7, Odometer: 400}, {DepthMm: 8, Odometer: 0}},
		"tiny wear":      {{DepthMm: 7.97, Odometer: 20000}, {DepthMm: 8, Odometer: 0}},
	} {
		store := &fakeTireWearStore{logs: logs, sessions: []models.TireMountSession{{MountedOdometer: 0}}}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), wearTire(models.TirePosStorage), 20000)
		if err != nil {
			t.Fatal(err)
		}
		if got.WearRatePer10kKm != 1.2 {
			t.Errorf("%s: expected fallback rate, got %v", name, got.WearRatePer10kKm)
		}
	}
}

func TestCalculateTireWearCondition(t *testing.T) {
	for _, tc := range []struct {
		depth float64
		want  string
	}{{2.5, "CRITICAL"}, {1.0, "CRITICAL"}, {2.6, "WARNING"}, {4.0, "WARNING"}, {4.1, "GOOD"}} {
		store := &fakeTireWearStore{logs: []models.TireLog{{DepthMm: tc.depth}}}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), wearTire(models.TirePosStorage), 0)
		if err != nil {
			t.Fatal(err)
		}
		if got.Condition != tc.want {
			t.Errorf("depth %v: expected %s, got %s", tc.depth, tc.want, got.Condition)
		}
	}
}

func TestCalculateTireWearMountedRunAndLifespan(t *testing.T) {
	odo := 20000.0
	tire := wearTire(models.TirePosFL)
	tire.MountedOdometer = &odo
	tire.AccumulatedDistanceKm = 5000
	tire.EstimatedLifespanKm = 20000
	tire.PurchasePrice = 20000 // cents
	store := &fakeTireWearStore{sessions: []models.TireMountSession{{MountedOdometer: odo}}}
	got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentRunKm != 5000 || got.TotalDistanceKm != 10000 {
		t.Errorf("unexpected run/total: %v / %v", got.CurrentRunKm, got.TotalDistanceKm)
	}
	if got.LifeProgressPct != 50 {
		t.Errorf("expected 50%% life progress, got %v", got.LifeProgressPct)
	}
	if got.CostPerKm != 0.01 {
		t.Errorf("expected 0.01 per km, got %v", got.CostPerKm)
	}
	if got.Sessions[0].DistanceKm != 5000 {
		t.Errorf("the open session should show the live run, got %v", got.Sessions[0].DistanceKm)
	}

	// An odometer before the mount reading adds no run; the progress is capped at 100 %.
	over, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if over.CurrentRunKm != 0 {
		t.Errorf("expected no run, got %v", over.CurrentRunKm)
	}
	tire.AccumulatedDistanceKm = 90000
	capped, _ := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
	if capped.LifeProgressPct != 100 {
		t.Errorf("expected progress capped at 100, got %v", capped.LifeProgressPct)
	}
}

func TestCalculateTireWearTelemetryRanges(t *testing.T) {
	odo := 100.0
	end := 900.0

	// Closed and open sessions are forwarded as odometer windows.
	store := &fakeTireWearStore{sessions: []models.TireMountSession{
		{MountedOdometer: 0, DismountedOdometer: &end},
		{MountedOdometer: odo},
	}}
	tire := wearTire(models.TirePosFL)
	tire.MountedOdometer = &odo
	if _, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 1000); err != nil {
		t.Fatal(err)
	}
	if len(store.gotRanges) != 2 || store.gotRanges[0].Max == nil || store.gotRanges[1].Max != nil {
		t.Errorf("unexpected ranges: %+v", store.gotRanges)
	}

	// Without sessions, a mounted tire falls back to its own mount odometer.
	store = &fakeTireWearStore{}
	if _, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 1000); err != nil {
		t.Fatal(err)
	}
	if len(store.gotRanges) != 1 || store.gotRanges[0].Min != odo || store.gotRanges[0].Max != nil {
		t.Errorf("unexpected fallback range: %+v", store.gotRanges)
	}

	// No vehicle: no telemetry lookup.
	store = &fakeTireWearStore{}
	noVehicle := &models.Tire{ID: "t2", CurrentPosition: models.TirePosFL, MountedOdometer: &odo}
	if _, err := NewTireWearService(store).CalculateTireWear(context.Background(), noVehicle, 1000); err != nil {
		t.Fatal(err)
	}
	if store.calls != 0 {
		t.Errorf("telemetry must not be queried without a vehicle")
	}

	// A telemetry failure degrades to the static estimate instead of failing.
	store = &fakeTireWearStore{telemetry: errors.New("db down"), sessions: []models.TireMountSession{{MountedOdometer: odo}}}
	got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 1000)
	if err != nil {
		t.Fatalf("telemetry error must not fail the calculation: %v", err)
	}
	if got.DrivesCount != 0 || got.WearExplanation != nil {
		t.Errorf("expected no dynamic data after a telemetry error: %+v", got)
	}
}

func TestCalculateTireWearDrivingStyles(t *testing.T) {
	odo := 0.0
	for _, tc := range []struct {
		name                    string
		pos                     models.TirePosition
		powerMax, powerMin, kwh float64
		style                   string
		stress                  float64
	}{
		{"gentle front", models.TirePosFR, 20, -10, 12, "ECO", 0},
		{"balanced any axle", models.TirePosStorage, 80, -35, 16, "BALANCED", 1.03},
		{"aggressive rear", models.TirePosRL, 200, -90, 22, "SPORT", 0},
		{"no power data front", models.TirePosFL, 0, 0, 0, "BALANCED", 1.03},
	} {
		tire := wearTire(tc.pos)
		tire.MountedOdometer = &odo
		store := &fakeTireWearStore{
			powerMax: tc.powerMax, powerMin: tc.powerMin, consumption: tc.kwh, drives: 5,
			sessions: []models.TireMountSession{{MountedOdometer: 0}},
		}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if got.DrivingStyle != tc.style {
			t.Errorf("%s: expected style %s, got %s (stress %v)", tc.name, tc.style, got.DrivingStyle, got.DrivingStressIndex)
		}
		if tc.stress != 0 && got.DrivingStressIndex != tc.stress {
			t.Errorf("%s: expected stress %v, got %v", tc.name, tc.stress, got.DrivingStressIndex)
		}
		if got.WearExplanation == nil || got.DrivesCount != 5 {
			t.Errorf("%s: expected an explanation for %d drives", tc.name, got.DrivesCount)
		}
	}
}

// The same driving gives the same index on every axle: a rotated set wears evenly.
func TestCalculateTireWearStressIgnoresTheCurrentPosition(t *testing.T) {
	var idx []float64
	for _, pos := range []models.TirePosition{models.TirePosFL, models.TirePosRR, models.TirePosStorage} {
		odo := 0.0
		tire := wearTire(pos)
		tire.MountedOdometer = &odo
		store := &fakeTireWearStore{powerMax: 120, powerMin: -60, consumption: 18, drives: 5, sessions: []models.TireMountSession{{MountedOdometer: 0, Position: pos}}}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 1000)
		if err != nil {
			t.Fatal(err)
		}
		idx = append(idx, got.DrivingStressIndex)
	}
	if idx[0] != idx[1] || idx[1] != idx[2] {
		t.Fatalf("stress index depends on the position: %v", idx)
	}
}

func TestPastAxleWeight(t *testing.T) {
	odo := func(v float64) *float64 { return &v }
	sessions := []models.TireMountSession{
		{MountedOdometer: 0, DismountedOdometer: odo(10000), Position: models.TirePosFL},
		{MountedOdometer: 10000, Position: models.TirePosRL},
	}
	// 10 000 km on the front axle, then 10 000 km on the rear axle.
	if got, want := pastAxleWeight(sessions, 0, 20000, models.TirePosRL), (frontAxleWeight+rearAxleWeight)/2; math.Abs(got-want) > 1e-9 {
		t.Errorf("mixed history: got %v want %v", got, want)
	}
	// Only the window between two readings counts.
	if got := pastAxleWeight(sessions, 12000, 20000, models.TirePosRL); got != rearAxleWeight {
		t.Errorf("window on the rear axle: got %v", got)
	}
	// No session in the window: the current position stands in.
	if got := pastAxleWeight(nil, 0, 20000, models.TirePosFR); got != frontAxleWeight {
		t.Errorf("no session: got %v", got)
	}
}

// A measured rate keeps its history: a tire that wore on the rear axle and is now at the front projects the
// measured rate scaled by the rotated weight over the rear weight, whatever its position today.
func TestCalculateTireWearMeasuredRateIsNormalisedByPastAxles(t *testing.T) {
	run := func(pos models.TirePosition, past models.TirePosition) float64 {
		odo := 0.0
		tire := wearTire(pos)
		tire.MountedOdometer = &odo
		store := &fakeTireWearStore{
			logs:     []models.TireLog{{DepthMm: 6.8, Odometer: 10000}, {DepthMm: 8, Odometer: 0}},
			powerMax: 80, powerMin: -35, consumption: 16, drives: 5,
			sessions: []models.TireMountSession{{MountedOdometer: 0, Position: past}},
		}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
		if err != nil {
			t.Fatal(err)
		}
		if got.WearRateSource != "measured" {
			t.Fatalf("wear source %q", got.WearRateSource)
		}
		return got.DynamicRemainingKm
	}
	rear, front := run(models.TirePosRL, models.TirePosRL), run(models.TirePosFL, models.TirePosFL)
	// Same measured wear: the rear history is more severe, so the rear tire gains more by rotating than the front one loses.
	if rear <= front {
		t.Fatalf("rear-worn tire should project longer once rotated: rear %v front %v", rear, front)
	}
	// The past axles set the neutral rate; the current position only sets which axle comes first.
	if other := run(models.TirePosFL, models.TirePosRL); math.Abs(other-rear)/rear > 0.05 {
		t.Fatalf("projection must follow the past axles, not the current position: %v vs %v", other, rear)
	}
}

func TestProjectRemainingKm(t *testing.T) {
	cases := []struct {
		name   string
		depth  float64
		start  models.TirePosition
		first  float64
		wantKm float64
	}{
		// Even number of periods: 10 000 km at the rear (1.15 mm) then 10 000 km at the front (0.92 mm).
		{"two periods from the rear", 2.07, models.TirePosRL, 10000, 20000},
		{"two periods from the front", 2.07, models.TirePosFR, 10000, 20000},
		// Odd number: one more period on the starting axle.
		{"rear then half a front period", 1.61, models.TirePosRL, 10000, 15000},
		{"front then rear", 1.61, models.TirePosFL, 10000, 16000},
		// A rotation done 6 000 km ago: 4 000 km left on the current axle (rear, 0.46 mm), then the front axle.
		{"first period shortened", 1.38, models.TirePosRR, 4000, 14000},
		// Rotation due now: the tire goes to the other axle at once.
		{"rotation due", 0.92, models.TirePosRL, 0, 10000},
		// Off a wheel: average weight of the set.
		{"storage", 1.035, models.TirePosStorage, 10000, 10000},
	}
	for _, c := range cases {
		if got := projectRemainingKm(c.depth, 1.0, c.start, c.first); math.Abs(got-c.wantKm) > 1 {
			t.Errorf("%s: got %.1f km, want %.1f", c.name, got, c.wantKm)
		}
	}
	if projectRemainingKm(0, 1, models.TirePosFL, 10000) != 0 || projectRemainingKm(2, 0, models.TirePosFL, 10000) != 0 {
		t.Error("no depth or no wear rate gives no projection")
	}
}

// A set that has been on its axle for more than a rotation interval is overdue: the projection swaps it to the other
// axle at once, exactly as for a rotation due now, and a longer run changes nothing more.
func TestCalculateTireWearOverdueRotationSwapsAtOnce(t *testing.T) {
	run := func(mountedAt float64) *TireWearStats {
		tire := wearTire(models.TirePosRL)
		tire.MountedOdometer = &mountedAt
		store := &fakeTireWearStore{
			logs:     []models.TireLog{{DepthMm: 6.8, Odometer: 10000}, {DepthMm: 8, Odometer: 0}},
			powerMax: 80, powerMin: -35, consumption: 16, drives: 5,
			sessions: []models.TireMountSession{{MountedOdometer: mountedAt, Position: models.TirePosRL}},
		}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, 10000)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	due, overdue := run(0), run(-15000)
	if overdue.CurrentRunKm <= tireRotationIntervalKm {
		t.Fatalf("setup: run %v km must exceed the rotation interval", overdue.CurrentRunKm)
	}
	if math.Abs(due.DynamicRemainingKm-overdue.DynamicRemainingKm) > 1 {
		t.Fatalf("overdue rotation should project like one due now: %v vs %v", overdue.DynamicRemainingKm, due.DynamicRemainingKm)
	}
}

// A few tenths of a millimetre must not drive the projection: two tires with the same distance and 0.4 mm versus
// 1.15 mm of measured wear project much closer than their raw rates (a factor 2.9) would give.
func TestCalculateTireWearSmallWearIsBlendedWithTheAverage(t *testing.T) {
	run := func(depth float64) *TireWearStats {
		store := &fakeTireWearStore{
			logs:     []models.TireLog{{DepthMm: depth, Odometer: 10000}, {DepthMm: 6.5, Odometer: 0}},
			sessions: []models.TireMountSession{{MountedOdometer: 0}},
		}
		got, err := NewTireWearService(store).CalculateTireWear(context.Background(), wearTire(models.TirePosStorage), 10000)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	front, rear := run(6.1), run(5.35)
	if front.WearRateConfidence >= 0.5 || rear.WearRateConfidence <= 0.5 {
		t.Errorf("confidence should follow the wear: front %v rear %v", front.WearRateConfidence, rear.WearRateConfidence)
	}
	if ratio := rear.WearRatePer10kKm / front.WearRatePer10kKm; ratio > 1.6 {
		t.Errorf("blended rates should stay close, got a ratio of %v", ratio)
	}
	if ratio := front.EstimatedRemainingKm / rear.EstimatedRemainingKm; ratio > 2 {
		t.Errorf("remaining km should stay within a factor 2, got %v", ratio)
	}
	// A large wear is almost entirely the measured rate.
	big := run(2.5)
	if big.WearRateConfidence < 0.8 {
		t.Errorf("4 mm worn should be mostly measured, got %v", big.WearRateConfidence)
	}
}
