package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationOdometerCountsTheReadingsAndEstimatesADay(t *testing.T) {
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 12, 0, 0, 0, time.UTC) }
	now := day(10, 19)
	anchors := []services.OdometerAnchor{
		{Date: day(1, 1), Km: 10000}, {Date: day(3, 1), Km: 12000}, {Date: now, Km: 15000},
	}
	at := day(2, 1)

	out := buildIntegrationOdometer("v1", 15000.4, anchors, now, &at)

	if out.Current.Value != 15000 || out.Readings != 2 || out.FirstReading == nil || *out.FirstReading != "2026-01-01" || *out.LastReading != "2026-03-01" {
		t.Fatalf("readings: %+v", out)
	}
	if out.Estimate == nil || out.Estimate.Source != services.OdometerEstimateInterpolated || out.Estimate.Date != "2026-02-01" || out.Estimate.Odometer.Value < 10000 || out.Estimate.Odometer.Value > 12000 {
		t.Errorf("a day between two readings is interpolated: %+v", out.Estimate)
	}

	none := buildIntegrationOdometer("v1", 0, nil, now, &at)
	if none.Readings != 0 || none.FirstReading != nil || none.Estimate != nil {
		t.Errorf("no reading, no estimate: %+v", none)
	}
	if noDay := buildIntegrationOdometer("v1", 1, anchors, now, nil); noDay.Estimate != nil {
		t.Errorf("no day asked, no estimate: %+v", noDay.Estimate)
	}
}
