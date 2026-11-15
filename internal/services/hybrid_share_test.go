package services

import (
	"math"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

const (
	hsElec = 16.0 // kWh/100 km, plug side
	hsFuel = 6.0  // L/100 km on the fuel part
)

var hsNow = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

func f64(v float64) *float64 { return &v }
func i32(v int) *int         { return &v }

type hsPeriod struct {
	km    float64
	share float64
}

type hsOpts struct {
	startDaysAgo  int  // age of the first fill-up
	noKwhUsed     bool // synchronised charges only carry the battery side
	manualCharges bool // charges typed by hand: kwh_added is the plug side, kwh_used is never stored
	chargeNoOdo   bool // charges carry no odometer
	liters        func(i int, l float64) *float64
	skipEndOdo    int // 1-based period whose closing fill-up has no odometer (0: none)
	partialMiddle bool
}

// hsSeries builds fill-ups and charges that follow the model: each period has a known share.
func hsSeries(periods []hsPeriod, o hsOpts) ([]models.FuelLog, []models.ChargeLog) {
	odo := 10000.0
	day := hsNow.AddDate(0, 0, -o.startDaysAgo)
	fuel := []models.FuelLog{{Date: day, Odometer: f64(odo), Liters: f64(40), IsFullTank: true}}
	var charges []models.ChargeLog
	for i, p := range periods {
		elecKm := p.km * p.share
		kwh := elecKm * hsElec / 100
		litres := (p.km - elecKm) * hsFuel / 100
		mid := day.AddDate(0, 0, 5)
		if kwh > 0 {
			c := models.ChargeLog{Date: mid, KwhAdded: kwh * 0.9, KwhUsed: f64(kwh)}
			if o.noKwhUsed {
				c.KwhUsed, c.KwhAdded = nil, kwh
			}
			if o.manualCharges {
				c.KwhUsed, c.KwhAdded, c.IsManual = nil, kwh, true
			}
			if !o.chargeNoOdo {
				c.Odometer = f64(odo + p.km/2)
			}
			charges = append(charges, c)
		}
		if o.partialMiddle && litres > 2 {
			fuel = append(fuel, models.FuelLog{Date: mid, Odometer: f64(odo + p.km/2), Liters: f64(1), IsFullTank: false})
			litres--
		}
		odo += p.km
		day = day.AddDate(0, 0, 10)
		l := f64(litres)
		if o.liters != nil {
			l = o.liters(i, litres)
		}
		end := models.FuelLog{Date: day, Odometer: f64(odo), Liters: l, IsFullTank: true}
		if o.skipEndOdo == i+1 {
			end.Odometer = nil
		}
		fuel = append(fuel, end)
	}
	return fuel, charges
}

func hsRun(fuel []models.FuelLog, charges []models.ChargeLog, ref *float64) HybridShareResult {
	return ComputeHybridShare(HybridShareInput{Fuel: fuel, Charges: charges, ReferenceKwh100: ref, Now: hsNow})
}

func near(t *testing.T, got *float64, want, tol float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("got nil, want %v", want)
	}
	if math.Abs(*got-want) > tol {
		t.Fatalf("got %v, want %v ± %v", *got, want, tol)
	}
}

func steady(n int, km, share float64) []hsPeriod {
	p := make([]hsPeriod, n)
	for i := range p {
		p[i] = hsPeriod{km, share}
	}
	return p
}

func TestHybridShareWithReference(t *testing.T) {
	tests := []struct {
		name      string
		periods   []hsPeriod
		opts      hsOpts
		wantShare float64
		wantLevel string
		tweak     func(f []models.FuelLog, c []models.ChargeLog)
		check     func(t *testing.T, r HybridShareResult)
	}{
		{name: "exact series", periods: steady(6, 300, 0.4), opts: hsOpts{startDaysAgo: 80}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Method != HybridMethodReference {
					t.Fatalf("method %q", r.Method)
				}
				near(t, r.FuelL100kmFuelPart, hsFuel, 0.01)
				near(t, r.ElectricKwh100km, hsElec, 0.01)
				if r.Quality.CoherenceGap > 0.01 {
					t.Fatalf("gap %v", r.Quality.CoherenceGap)
				}
			}},
		{name: "weighted by distance, not averaged", periods: []hsPeriod{{600, 0.9}, {100, 0.1}, {100, 0.1}, {100, 0.1}, {100, 0.1}, {100, 0.1}}, opts: hsOpts{startDaysAgo: 80},
			wantShare: (540 + 5*10.0) / 1100, wantLevel: HybridLevelGood},
		{name: "never electric", periods: steady(6, 300, 0), opts: hsOpts{startDaysAgo: 80}, wantShare: 0, wantLevel: HybridLevelGood},
		{name: "a partial fill-up inside a period changes nothing", periods: steady(6, 300, 0.5), opts: hsOpts{startDaysAgo: 80, partialMiddle: true}, wantShare: 0.5, wantLevel: HybridLevelGood},
		{name: "charges without odometer follow their date", periods: steady(6, 300, 0.25), opts: hsOpts{startDaysAgo: 80, chargeNoOdo: true}, wantShare: 0.25, wantLevel: HybridLevelGood},
		{name: "battery side only is flagged and lowers the level", periods: steady(4, 300, 0.4), opts: hsOpts{startDaysAgo: 40, noKwhUsed: true}, wantShare: 0.4, wantLevel: HybridLevelLow,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.LessPreciseEnergy != 4 {
					t.Fatalf("less precise %d", r.Quality.LessPreciseEnergy)
				}
			}},
		{name: "manual charges are read on the plug side and are not flagged", periods: steady(6, 300, 0.4), opts: hsOpts{startDaysAgo: 80, manualCharges: true}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.LessPreciseEnergy != 0 {
					t.Fatalf("less precise %d", r.Quality.LessPreciseEnergy)
				}
			}},
		{name: "three periods are enough, but low", periods: steady(3, 300, 0.4), opts: hsOpts{startDaysAgo: 30}, wantShare: 0.4, wantLevel: HybridLevelLow},
		{name: "two periods are not", periods: steady(2, 400, 0.4), opts: hsOpts{startDaysAgo: 20}, wantLevel: HybridLevelInsufficient,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.InsufficientBecause != "not_enough_data" || r.Suggested != nil {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "enough periods but too few kilometres", periods: steady(5, 60, 0.4), opts: hsOpts{startDaysAgo: 50}, wantLevel: HybridLevelInsufficient},
		{name: "a missing odometer between two others is interpolated", periods: steady(7, 300, 0.4), opts: hsOpts{startDaysAgo: 80, skipEndOdo: 3}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.DiscardedNoOdometer != 0 || r.Quality.SegmentsUsed != 7 {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "a last fill-up without odometer cannot be estimated and is discarded", periods: steady(7, 300, 0.4), opts: hsOpts{startDaysAgo: 80, skipEndOdo: 7}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.DiscardedNoOdometer != 1 || r.Quality.SegmentsUsed != 6 {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "a period without litres is discarded", periods: steady(7, 300, 0.4),
			opts: hsOpts{startDaysAgo: 80, liters: func(i int, l float64) *float64 {
				if i == 2 {
					return nil
				}
				return f64(l)
			}}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.DiscardedNoLiters != 1 || r.Quality.SegmentsDiscarded != 1 {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "a very short period is discarded", periods: append(steady(6, 300, 0.4), hsPeriod{10, 0.4}), opts: hsOpts{startDaysAgo: 90}, wantShare: 0.4, wantLevel: HybridLevelGood,
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.DiscardedTooShort != 1 {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "an impossible period is discarded", periods: steady(7, 300, 0.4), opts: hsOpts{startDaysAgo: 90}, wantShare: 0.4, wantLevel: HybridLevelGood,
			tweak: func(_ []models.FuelLog, c []models.ChargeLog) { c[6].KwhUsed = f64(*c[6].KwhUsed * 3) },
			check: func(t *testing.T, r HybridShareResult) {
				if r.Quality.DiscardedIncoherent != 1 {
					t.Fatalf("%+v", r.Quality)
				}
			}},
		{name: "almost only electric", periods: steady(6, 300, 0.98), opts: hsOpts{startDaysAgo: 80}, wantShare: 0.98, wantLevel: HybridLevelGood},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fuel, charges := hsSeries(tc.periods, tc.opts)
			if tc.tweak != nil {
				tc.tweak(fuel, charges)
			}
			r := hsRun(fuel, charges, f64(hsElec))
			if r.Quality.Level != tc.wantLevel {
				t.Fatalf("level %q, want %q (%+v)", r.Quality.Level, tc.wantLevel, r.Quality)
			}
			if tc.wantLevel != HybridLevelInsufficient {
				near(t, r.Suggested, tc.wantShare, 0.005)
			}
			if tc.check != nil {
				tc.check(t, r)
			}
		})
	}
}

func TestHybridShareWindow(t *testing.T) {
	// Old periods at 80 % then recent ones at 20 %: the last twelve months only speak for the recent ones.
	old, _ := hsSeries(steady(6, 300, 0.8), hsOpts{startDaysAgo: 900})
	oldCharges := func() []models.ChargeLog { _, c := hsSeries(steady(6, 300, 0.8), hsOpts{startDaysAgo: 900}); return c }()
	// recent series continues the odometer after the old one
	recent, recentCharges := hsSeries(steady(4, 300, 0.2), hsOpts{startDaysAgo: 45})
	shift := 10000.0 + 6*300 - 10000.0
	for i := range recent {
		recent[i].Odometer = f64(*recent[i].Odometer + shift)
	}
	for i := range recentCharges {
		recentCharges[i].Odometer = f64(*recentCharges[i].Odometer + shift)
	}
	r := hsRun(append(old, recent...), append(oldCharges, recentCharges...), f64(hsElec))
	if r.Quality.WindowMonths != hybridWindowMonths {
		t.Fatalf("window %d", r.Quality.WindowMonths)
	}
	near(t, r.Suggested, 0.2, 0.01)

	// Only two recent periods: the whole history is used.
	recent2, recentCharges2 := hsSeries(steady(2, 300, 0.2), hsOpts{startDaysAgo: 25})
	for i := range recent2 {
		recent2[i].Odometer = f64(*recent2[i].Odometer + shift)
	}
	for i := range recentCharges2 {
		recentCharges2[i].Odometer = f64(*recentCharges2[i].Odometer + shift)
	}
	r = hsRun(append(append([]models.FuelLog(nil), old...), recent2...), append(append([]models.ChargeLog(nil), oldCharges...), recentCharges2...), f64(hsElec))
	if r.Quality.WindowMonths != 0 {
		t.Fatalf("window %d", r.Quality.WindowMonths)
	}
	if r.Suggested == nil || *r.Suggested < 0.5 {
		t.Fatalf("whole history should weigh the old periods: %v", r.Suggested)
	}
}

func TestHybridShareRegression(t *testing.T) {
	varied := []hsPeriod{{300, 0.1}, {300, 0.7}, {300, 0.3}, {300, 0.9}, {300, 0.5}, {300, 0.2}, {300, 0.6}}
	t.Run("recovers both consumptions without a reference", func(t *testing.T) {
		fuel, charges := hsSeries(varied, hsOpts{startDaysAgo: 80})
		r := hsRun(fuel, charges, nil)
		if r.Method != HybridMethodRegression {
			t.Fatalf("method %q (%+v)", r.Method, r.Quality)
		}
		near(t, r.Suggested, 0.471, 0.01)
		near(t, r.ElectricKwh100km, hsElec, 0.2)
		near(t, r.FuelL100kmFuelPart, hsFuel, 0.2)
	})
	t.Run("energy and litres moving together cannot be separated", func(t *testing.T) {
		// the same share every period: kWh and litres are proportional
		fuel, charges := hsSeries(steady(7, 300, 0.4), hsOpts{startDaysAgo: 80})
		r := hsRun(fuel, charges, nil)
		if r.Suggested != nil || r.Quality.InsufficientBecause != "energy_and_fuel_not_separable" {
			t.Fatalf("%+v %v", r.Quality, r.Suggested)
		}
	})
	t.Run("no energy at all cannot be separated", func(t *testing.T) {
		fuel, charges := hsSeries(steady(7, 300, 0), hsOpts{startDaysAgo: 80})
		if r := hsRun(fuel, charges, nil); r.Suggested != nil {
			t.Fatalf("%+v", r)
		}
	})
}

func TestHybridShareBatteryPercentages(t *testing.T) {
	// Each period holds two charges with percentages. 40 kWh usable battery.
	const capKwh = 40.0
	build := func(share float64) ([]models.FuelLog, []models.ChargeLog) {
		fuel, _ := hsSeries(steady(6, 400, share), hsOpts{startDaysAgo: 80})
		var charges []models.ChargeLog
		odo := 10000.0
		day := hsNow.AddDate(0, 0, -80)
		for i := 0; i < 6; i++ {
			elecKm := 400 * share
			used := elecKm * hsElec / 100 // kWh drawn from the battery over the period
			// first charge to 90 %, 100 km into the period; the last starts after 'used' minus nothing in between
			// (no charge between them): start of the last = 90 % minus what the battery gave in the 200 km between
			between := used * 200 / 400
			first := models.ChargeLog{Date: day.AddDate(0, 0, 2), Odometer: f64(odo + 100), StartBatteryLevel: i32(40), EndBatteryLevel: i32(90), KwhAdded: capKwh * 0.5, KwhUsed: f64(capKwh * 0.5 / 0.9)}
			startLast := 90 - int(math.Round(between/capKwh*100))
			last := models.ChargeLog{Date: day.AddDate(0, 0, 6), Odometer: f64(odo + 300), StartBatteryLevel: i32(startLast), EndBatteryLevel: i32(90), KwhAdded: capKwh * float64(90-startLast) / 100, KwhUsed: f64(capKwh * float64(90-startLast) / 100 / 0.9)}
			charges = append(charges, first, last)
			odo += 400
			day = day.AddDate(0, 0, 10)
		}
		return fuel, charges
	}
	fuel, charges := build(0.5)
	r := hsRun(fuel, charges, f64(hsElec))
	near(t, r.BatteryKwh, capKwh, 0.1)
	if !r.Quality.CapacityFromPercents || r.Quality.SocCoverage != 1 {
		t.Fatalf("%+v", r.Quality)
	}
	near(t, r.Suggested, 0.5, 0.03)

	t.Run("charges spanning too few points give no capacity", func(t *testing.T) {
		_, c := build(0.5)
		for i := range c {
			c[i].StartBatteryLevel = i32(*c[i].EndBatteryLevel - 10)
		}
		r := hsRun(fuel, c, f64(hsElec))
		if r.BatteryKwh != nil || r.Quality.CapacityFromPercents || r.Quality.SocCoverage != 0 {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("a period with a single charge keeps the plug-side sum", func(t *testing.T) {
		_, c := build(0.5)
		var kept []models.ChargeLog
		for i, ch := range c {
			if i%2 == 0 {
				kept = append(kept, ch)
			}
		}
		r := hsRun(fuel, kept, f64(hsElec))
		if r.Quality.SocCoverage != 0 {
			t.Fatalf("%+v", r.Quality)
		}
	})
	t.Run("charges without a percentage are not windows", func(t *testing.T) {
		_, c := build(0.5)
		c[1].EndBatteryLevel = nil
		r := hsRun(fuel, c, f64(hsElec))
		if r.Quality.SocCoverage >= 1 {
			t.Fatalf("%+v", r.Quality)
		}
	})
	t.Run("a window with a negative energy is ignored", func(t *testing.T) {
		_, c := build(0.5)
		for i := range c {
			if i%2 == 1 {
				c[i].StartBatteryLevel = i32(100) // the battery 'gained' charge while driving
			}
		}
		r := hsRun(fuel, c, f64(hsElec))
		if r.Quality.SocCoverage != 0 {
			t.Fatalf("%+v", r.Quality)
		}
	})
	t.Run("two charges too close do not make a window", func(t *testing.T) {
		_, c := build(0.5)
		for i := 1; i < len(c); i += 2 {
			c[i].Odometer = f64(*c[i-1].Odometer + 5)
		}
		r := hsRun(fuel, c, f64(hsElec))
		if r.Quality.SocCoverage != 0 {
			t.Fatalf("%+v", r.Quality)
		}
	})
}

func TestHybridShareNoData(t *testing.T) {
	r := hsRun(nil, nil, f64(hsElec))
	if r.Suggested != nil || r.Quality.Level != HybridLevelInsufficient {
		t.Fatalf("%+v", r)
	}
	// A reference of zero is no reference.
	fuel, charges := hsSeries(varyShare(), hsOpts{startDaysAgo: 80})
	if r := hsRun(fuel, charges, f64(0)); r.Method != HybridMethodRegression {
		t.Fatalf("method %q", r.Method)
	}
}

func varyShare() []hsPeriod {
	return []hsPeriod{{300, 0.1}, {300, 0.7}, {300, 0.3}, {300, 0.9}, {300, 0.5}, {300, 0.2}, {300, 0.6}}
}

func TestHybridRegressionNonNegative(t *testing.T) {
	seg := func(kwh, l, km float64) *hybridSegment { return &hybridSegment{kwh: kwh, liters: l, km: km} }
	t.Run("a negative electric coefficient is dropped", func(t *testing.T) {
		used := []*hybridSegment{seg(10, 30, 550), seg(20, 25, 400), seg(30, 20, 250), seg(5, 40, 775)}
		var res HybridShareResult
		if !hybridRegression(used, &res) {
			t.Fatal("refused")
		}
		for _, s := range used {
			if s.elecKm != 0 {
				t.Fatalf("electric distance %v", s.elecKm)
			}
		}
		if res.ElectricKwh100km != nil {
			t.Fatalf("%v", *res.ElectricKwh100km)
		}
	})
	t.Run("a negative fuel coefficient is dropped", func(t *testing.T) {
		used := []*hybridSegment{seg(10, 5, 130), seg(20, 10, 260), seg(30, 2, 442), seg(5, 8, 43)}
		var res HybridShareResult
		if !hybridRegression(used, &res) {
			t.Fatal("refused")
		}
		// the fit keeps the energy only, capped by the distance of each period
		for _, s := range used {
			if s.elecKm <= 0 || s.elecKm > s.km {
				t.Fatalf("electric distance %v of %v km", s.elecKm, s.km)
			}
		}
		if res.ElectricKwh100km == nil {
			t.Fatal("the electric consumption should be reported")
		}
	})
	t.Run("no gap without fuel", func(t *testing.T) {
		if g := hybridCoherenceGap([]*hybridSegment{{km: 100, elecKm: 100, liters: 3}}, HybridShareResult{}); g != 0 {
			t.Fatalf("%v", g)
		}
	})
}

func TestHybridShareBatteryWindowWithAMiddleCharge(t *testing.T) {
	const capKwh = 40.0
	fuel, _ := hsSeries(steady(6, 400, 0.5), hsOpts{startDaysAgo: 80})
	var charges []models.ChargeLog
	odo := 10000.0
	day := hsNow.AddDate(0, 0, -80)
	for i := 0; i < 6; i++ {
		mk := func(off float64, start, end int, d int) models.ChargeLog {
			added := capKwh * float64(end-start) / 100
			return models.ChargeLog{Date: day.AddDate(0, 0, d), Odometer: f64(odo + off), StartBatteryLevel: i32(start), EndBatteryLevel: i32(end), KwhAdded: added, KwhUsed: f64(added / 0.9)}
		}
		charges = append(charges, mk(100, 40, 90, 1), mk(200, 70, 90, 3), mk(300, 70, 90, 5))
		odo += 400
		day = day.AddDate(0, 0, 10)
	}
	r := hsRun(fuel, charges, f64(hsElec))
	if r.Quality.SocCoverage != 1 {
		t.Fatalf("%+v", r.Quality)
	}
	near(t, r.Suggested, 0.5, 0.02)
}
