package services

import (
	"context"
	"math"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestTireReadingFallbackWithoutMountSessions(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		reading, current, consumed float64
	}{
		{"reading on current mount", 25000, 30000, 5000},
		{"reading before current mount", 15000, 30000, 10000},
		{"reading ahead of odometer", 35000, 30000, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mount := 20000.0
			tire := wearTire(models.TirePosFL)
			tire.MountedOdometer = &mount
			tire.AccumulatedDistanceKm = 12000
			store := &fakeTireWearStore{logs: []models.TireLog{{DepthMm: 6, Odometer: tc.reading}}}
			stats, err := NewTireWearService(store).CalculateTireWear(context.Background(), tire, tc.current)
			if err != nil {
				t.Fatal(err)
			}
			want := math.Round((6-stats.MinLegalDepthMm)/defaultWearRatePer10k*10000 - tc.consumed)
			if stats.EstimatedRemainingKm != want {
				t.Fatalf("remaining = %v, want %v", stats.EstimatedRemainingKm, want)
			}
			if stats.CurrentDepthMm != 6 {
				t.Fatal("recorded depth changed")
			}
		})
	}
}

func TestTireLifetimeDistanceAtReadingIncludesInitialMileage(t *testing.T) {
	tire := &models.Tire{InitialDistanceKm: 4000}
	end := 15000.0
	sessions := []models.TireMountSession{
		{MountedOdometer: 10000, DismountedOdometer: &end},
		{MountedOdometer: 25000},
	}
	if got := tireLifetimeDistanceAtReading(tire, sessions, 28000); got != 12000 {
		t.Fatalf("lifetime distance = %v, want 12000", got)
	}
}
