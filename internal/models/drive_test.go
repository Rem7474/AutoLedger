package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDriveIsHighway(t *testing.T) {
	ptrFloat := func(v float64) *float64 { return &v }
	ptrInt := func(v int) *int { return &v }

	tests := []struct {
		name       string
		distanceKm float64
		speedAvg   *float64
		speedMax   *int
		want       bool
	}{
		{
			name:       "nil speeds",
			distanceKm: 50,
			speedAvg:   nil,
			speedMax:   nil,
			want:       false,
		},
		{
			name:       "standard long fast highway drive",
			distanceKm: 42,
			speedAvg:   ptrFloat(72.5),
			speedMax:   ptrInt(115),
			want:       true,
		},
		{
			name:       "highway drive slowed down by traffic jam with vmax > 125",
			distanceKm: 35,
			speedAvg:   ptrFloat(52.0),
			speedMax:   ptrInt(131),
			want:       true,
		},
		{
			name:       "short ramp or urban hop with vmax > 125 under 20km threshold",
			distanceKm: 12,
			speedAvg:   ptrFloat(60.0),
			speedMax:   ptrInt(130),
			want:       false,
		},
		{
			name:       "highway drive exactly at 20km with vmax > 125",
			distanceKm: 20,
			speedAvg:   ptrFloat(65.0),
			speedMax:   ptrInt(126),
			want:       true,
		},
		{
			name:       "vmax exactly 125 does not trigger vmax rule alone",
			distanceKm: 25,
			speedAvg:   ptrFloat(60.0),
			speedMax:   ptrInt(125),
			want:       false,
		},
		{
			name:       "30 km on a 110 km/h stretch (rain or works) at highway average",
			distanceKm: 30,
			speedAvg:   ptrFloat(88.0),
			speedMax:   ptrInt(112),
			want:       true,
		},
		{
			name:       "30 km at 110 km/h top speed but slow average stays out",
			distanceKm: 30,
			speedAvg:   ptrFloat(55.0),
			speedMax:   ptrInt(112),
			want:       false,
		},
		{
			name:       "short 10 km hop between two exits",
			distanceKm: 10,
			speedAvg:   ptrFloat(92.0),
			speedMax:   ptrInt(118),
			want:       true,
		},
		{
			name:       "exactly 8 km hop at 105 km/h and 70 km/h average",
			distanceKm: 8,
			speedAvg:   ptrFloat(70.0),
			speedMax:   ptrInt(105),
			want:       true,
		},
		{
			name:       "under 8 km is never a highway drive",
			distanceKm: 7.5,
			speedAvg:   ptrFloat(95.0),
			speedMax:   ptrInt(120),
			want:       false,
		},
		{
			name:       "fast national road hop at 100 km/h top speed",
			distanceKm: 12,
			speedAvg:   ptrFloat(80.0),
			speedMax:   ptrInt(100),
			want:       false,
		},
		{
			name:       "secondary road 50km at 65 km/h",
			distanceKm: 50,
			speedAvg:   ptrFloat(65.0),
			speedMax:   ptrInt(85),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Drive{
				DistanceKm: kmPtr(tt.distanceKm),
				SpeedAvg:   tt.speedAvg,
				SpeedMax:   tt.speedMax,
			}
			if got := d.IsHighway(); got != tt.want {
				t.Errorf("Drive.IsHighway() = %v, want %v (dist=%.1f, avg=%v, max=%v)",
					got, tt.want, tt.distanceKm, tt.speedAvg, tt.speedMax)
			}
		})
	}
}

func TestDriveWithoutDistanceIsNullInJSONAndZeroForCalculations(t *testing.T) {
	d := Drive{}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"distance_km":null`) {
		t.Errorf("an unknown distance is null, not 0: %s", raw)
	}
	if d.Distance() != 0 || d.IsHighway() {
		t.Errorf("unknown distance: %v, highway %v", d.Distance(), d.IsHighway())
	}
	d.DistanceKm = kmPtr(42)
	if d.Distance() != 42 {
		t.Errorf("known distance: %v", d.Distance())
	}
}
