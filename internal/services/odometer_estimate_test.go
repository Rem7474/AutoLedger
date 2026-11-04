package services

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestEstimateOdometerAt(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	anchors := []OdometerAnchor{{day(11), 1100, false}, {day(1), 1000, false}, {day(21), 1500, false}}

	cases := []struct {
		name    string
		anchors []OdometerAnchor
		at      time.Time
		km      float64
		source  string
		ok      bool
	}{
		{"on a reading", anchors, day(11), 1100, OdometerEstimateExact, true},
		{"halfway between two readings", anchors, day(6), 1050, OdometerEstimateInterpolated, true},
		{"between the last two readings", anchors, day(16), 1300, OdometerEstimateInterpolated, true},
		{"before the first reading", anchors, day(1).AddDate(0, 0, -5), 1000, OdometerEstimateBounded, true},
		{"after the last reading", anchors, day(30), 1500, OdometerEstimateBounded, true},
		{"a lower reading later is ignored", []OdometerAnchor{{day(1), 1000, false}, {day(5), 900, false}, {day(11), 1100, false}}, day(6), 1050, OdometerEstimateInterpolated, true},
		{"zero readings are ignored", []OdometerAnchor{{day(1), 0, false}, {day(11), 1100, false}}, day(5), 1100, OdometerEstimateBounded, true},
		{"before the first reading, spread from the origin", []OdometerAnchor{{day(1), 0, true}, {day(11), 1000, false}}, day(6), 500, OdometerEstimateInterpolated, true},
		{"on the origin", []OdometerAnchor{{day(1), 0, true}, {day(11), 1000, false}}, day(1), 0, OdometerEstimateExact, true},
		{"before the origin", []OdometerAnchor{{day(11), 0, true}, {day(21), 1000, false}}, day(1), 0, OdometerEstimateBounded, true},
		{"an origin on the date of a reading is ignored", []OdometerAnchor{{day(1), 0, true}, {day(1), 800, false}, {day(11), 1000, false}}, day(1), 800, OdometerEstimateExact, true},
		{"only an origin", []OdometerAnchor{{day(1), 0, true}}, day(6), 0, OdometerEstimateBounded, true},
		{"no reading", nil, day(5), 0, "", false},
	}
	for _, c := range cases {
		km, source, ok := EstimateOdometerAt(c.anchors, c.at)
		if ok != c.ok || source != c.source || km < c.km-0.001 || km > c.km+0.001 {
			t.Errorf("%s: got %v %q %v, want %v %q %v", c.name, km, source, ok, c.km, c.source, c.ok)
		}
	}
}

// referenceEstimate is the estimate of the readings dated at or before cutoff, computed the direct way: filter, clean
// and sort the readings for each question. The curve must answer exactly the same.
func referenceEstimate(anchors []OdometerAnchor, cutoff *time.Time, at time.Time) (float64, string, bool) {
	var kept []OdometerAnchor
	for _, a := range anchors {
		if cutoff == nil || !a.Date.After(*cutoff) {
			kept = append(kept, a)
		}
	}
	var pts []OdometerAnchor
	for _, a := range kept {
		if a.Km > 0 || a.Origin {
			pts = append(pts, a)
		}
	}
	sort.SliceStable(pts, func(i, j int) bool {
		if pts[i].Date.Equal(pts[j].Date) {
			return pts[i].Km < pts[j].Km
		}
		return pts[i].Date.Before(pts[j].Date)
	})
	clean := pts[:0]
	for _, p := range pts {
		if len(clean) > 0 && p.Km < clean[len(clean)-1].Km {
			continue
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return 0, "", false
	}
	if len(clean) > 1 && clean[0].Origin && clean[0].Date.Equal(clean[1].Date) {
		clean = clean[1:]
	}
	if !at.After(clean[0].Date) {
		return clean[0].Km, boundedOrExact(clean[0].Date, at), true
	}
	last := clean[len(clean)-1]
	if !at.Before(last.Date) {
		return last.Km, boundedOrExact(last.Date, at), true
	}
	i := sort.Search(len(clean), func(i int) bool { return !clean[i].Date.Before(at) })
	next, prev := clean[i], clean[i-1]
	if next.Date.Equal(at) {
		return next.Km, OdometerEstimateExact, true
	}
	ratio := at.Sub(prev.Date).Seconds() / next.Date.Sub(prev.Date).Seconds()
	return prev.Km + ratio*(next.Km-prev.Km), OdometerEstimateInterpolated, true
}

// Readings that repeat a date, go backwards, sit at 0 km or start the vehicle's life: the curve built once, with or
// without a cutoff, answers like the estimate rebuilt from the readings for each question.
func TestOdometerCurveMatchesTheDirectEstimate(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return base.AddDate(0, 0, d) }
	for round := 0; round < 200; round++ {
		anchors := make([]OdometerAnchor, rng.Intn(40))
		for i := range anchors {
			anchors[i] = OdometerAnchor{Date: day(rng.Intn(60)), Km: float64(rng.Intn(5)) * 1000 * float64(rng.Intn(3)), Origin: rng.Intn(12) == 0}
			if rng.Intn(3) > 0 {
				anchors[i].Km += float64(i) * 10
			}
		}
		curve := NewOdometerCurve(anchors)
		for q := 0; q < 30; q++ {
			at := day(rng.Intn(80) - 10).Add(time.Duration(rng.Intn(3)) * 8 * time.Hour)
			cutoff := day(rng.Intn(80) - 10)
			for _, c := range []*time.Time{nil, &cutoff} {
				view := curve
				if c != nil {
					view = curve.Until(*c)
				}
				wantKm, wantSource, wantOK := referenceEstimate(anchors, c, at)
				gotKm, gotSource, gotOK := view.At(at)
				if gotKm != wantKm || gotSource != wantSource || gotOK != wantOK {
					t.Fatalf("round %d, cutoff %v, at %v: got (%v, %q, %v), want (%v, %q, %v)\nreadings: %+v", round, c, at, gotKm, gotSource, gotOK, wantKm, wantSource, wantOK, anchors)
				}
			}
		}
	}
}

func TestOdometerCurveDoesNotChangeTheReadingsItIsGiven(t *testing.T) {
	d := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	anchors := []OdometerAnchor{{Date: d.AddDate(0, 0, 2), Km: 300}, {Date: d, Km: 100}, {Date: d.AddDate(0, 0, 1), Km: 50}}
	before := append([]OdometerAnchor(nil), anchors...)
	NewOdometerCurve(anchors)
	if !reflect.DeepEqual(anchors, before) {
		t.Fatalf("readings were reordered: %+v", anchors)
	}
}
