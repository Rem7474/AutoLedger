package services

import (
	"context"
	"log/slog"
	"time"
)

// odometerDistanceKm is the distance the odometer says the vehicle covered between two instants, spread the way the
// monthly mileage smoothing does. A zero `from` starts at the first reading. It is 0 when the readings do not allow it.
func odometerDistanceKm(anchors []OdometerAnchor, from, to time.Time) float64 {
	start, _, ok := EstimateOdometerAt(anchors, from)
	if !ok {
		return 0
	}
	end, _, ok := EstimateOdometerAt(anchors, to)
	if !ok || end <= start {
		return 0
	}
	return end - start
}

// anchorsUpTo keeps the readings dated at or before t. The rates of a past date are read with the readings of that
// date only, so a reading entered later does not move them.
func anchorsUpTo(anchors []OdometerAnchor, t time.Time) []OdometerAnchor {
	kept := make([]OdometerAnchor, 0, len(anchors))
	for _, a := range anchors {
		if !a.Date.After(t) {
			kept = append(kept, a)
		}
	}
	return kept
}

// coveredDistanceKm is the denominator of a cost-per-km rate whose cost spans [from, to]. The recorded drives only
// count the trips the vehicle was fed (a vehicle tracked by hand, or fed by Home Assistant for a few weeks, has far
// fewer drives than kilometres), so a cost accumulated over years divided by them gives an absurd rate. The odometer
// covers the whole period: the larger of the two is used.
func coveredDistanceKm(tracked float64, anchors []OdometerAnchor, from, to time.Time) float64 {
	if odo := odometerDistanceKm(anchors, from, to); odo > tracked {
		return odo
	}
	return tracked
}

// odometerAnchors reads the vehicle's odometer history. A failure leaves the rates on the recorded drives alone.
func (s *CarpoolService) odometerAnchors(ctx context.Context, vehicleID string) []OdometerAnchor {
	stored, err := s.repo.ListOdometerAnchors(ctx, vehicleID)
	if err != nil {
		slog.Warn("unit rates: odometer history unavailable", "component", "carpool", "vehicle_id", vehicleID, "error", err)
		return nil
	}
	anchors := make([]OdometerAnchor, len(stored))
	for i, a := range stored {
		anchors[i] = OdometerAnchor{Date: a.Date, Km: a.Km, Origin: a.Origin}
	}
	return anchors
}
