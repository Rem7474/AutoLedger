package services

import (
	"sort"
	"time"
)

// OdometerAnchor is a known odometer reading: a drive end, a checkpoint, a fill-up, the start of the ownership.
type OdometerAnchor struct {
	Date time.Time
	Km   float64
	// Origin marks the start of the vehicle's life (an ownership recorded at 0 km, as the monthly mileage smoothing reads it): it counts
	// as 0 km, so a date before the first reading is spread from the age of the car instead of keeping that reading.
	Origin bool
}

// Sources of an odometer estimate.
const (
	OdometerEstimateExact        = "exact"
	OdometerEstimateInterpolated = "interpolated"
	OdometerEstimateBounded      = "bounded"
)

// OdometerCurve is the usable readings of a vehicle, cleaned and sorted once so that many dates can be estimated
// without reading the history again: a reading lower than an earlier one is dropped, and so is an origin that shares
// its date with a real reading (it would shadow it).
type OdometerCurve struct {
	pts []OdometerAnchor
}

// NewOdometerCurve cleans and sorts the readings. The slice it is given is left as it is.
func NewOdometerCurve(anchors []OdometerAnchor) OdometerCurve {
	pts := make([]OdometerAnchor, 0, len(anchors))
	for _, a := range anchors {
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
	if len(clean) > 1 && clean[0].Origin && clean[0].Date.Equal(clean[1].Date) {
		clean = clean[1:]
	}
	return OdometerCurve{pts: clean}
}

// Until is the curve of the readings dated at or before t: the readings of a past date, so that a reading entered
// later does not move what was estimated then. It shares the readings of the curve it comes from.
func (c OdometerCurve) Until(t time.Time) OdometerCurve {
	n := sort.Search(len(c.pts), func(i int) bool { return c.pts[i].Date.After(t) })
	return OdometerCurve{pts: c.pts[:n]}
}

// At estimates the odometer at a date the way the monthly mileage smoothing spreads the distance between two
// readings: linearly in time between the readings on either side. Before the first reading it spreads from the
// origin of the vehicle when there is one, else keeps that reading; after the last one it keeps that reading.
// ok is false when there is no usable reading.
func (c OdometerCurve) At(at time.Time) (km float64, source string, ok bool) {
	clean := c.pts
	if len(clean) == 0 {
		return 0, "", false
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

// EstimateOdometerAt estimates the odometer at one date from the readings. To estimate several dates, build the
// curve once with NewOdometerCurve.
func EstimateOdometerAt(anchors []OdometerAnchor, at time.Time) (km float64, source string, ok bool) {
	return NewOdometerCurve(anchors).At(at)
}

func boundedOrExact(anchor, at time.Time) string {
	if anchor.Equal(at) {
		return OdometerEstimateExact
	}
	return OdometerEstimateBounded
}
