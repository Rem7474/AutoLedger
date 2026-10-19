package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func TestIntegrationCarpoolNamesThePassengersAndTheirStops(t *testing.T) {
	label := func(s string) *string { return &s }
	trip := models.CarpoolTripWithPassengers{
		CarpoolTrip: models.CarpoolTrip{Title: "Weekend", Date: time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC), DistanceKm: 300, TotalCost: 3000, TotalRevenue: 1800, NetCost: 1200, Notes: label("  ")},
		Legs: []models.CarpoolLeg{
			{OrderIndex: 0, StartLabel: label("Lyon"), EndLabel: label("45.7, 4.8"), DistanceKm: 150, ElectricityCost: 1500},
			{OrderIndex: 1, StartLabel: label("45.7, 4.8"), EndLabel: label("Paris"), DistanceKm: 150, ElectricityCost: 1500},
		},
		Passengers: []models.CarpoolPassenger{
			{PassengerName: "Alice", Seats: 1, BoardStopIndex: 0, AlightStopIndex: 2, AmountPaid: 1800, CostShare: 1500, Balance: 300},
			{PassengerName: "Bob", Seats: 1, BoardStopIndex: 1, AlightStopIndex: 9, AmountPaid: 0},
		},
	}
	rate := 60.0
	out := buildIntegrationCarpool("v1", "CHF", []models.CarpoolTripWithPassengers{trip, trip}, models.CarpoolSummary{TotalTrips: 2, TotalPassengers: 4, TotalDistanceKm: 600, TotalRealCost: 6000, TotalRevenue: 3600, TotalNetCost: 2400, CoverageRatePct: &rate, NetCostPerKm: 4}, 1)

	if !out.Truncated || len(out.Trips) != 1 || out.Summary.Trips != 2 || out.Summary.Revenue.Amount != "36.00" || out.Summary.NetCostPerKm == nil {
		t.Fatalf("summary and cap: %+v", out)
	}
	got := out.Trips[0]
	if len(got.Stops) != 3 || got.Stops[0] == nil || *got.Stops[0] != "Lyon" || got.Stops[1] != nil || *got.Stops[2] != "Paris" {
		t.Errorf("a stop stored as coordinates is unknown: %v", got.Stops)
	}
	if got.Notes != nil {
		t.Errorf("a blank note is no note: %v", got.Notes)
	}
	alice, bob := got.Passengers[0], got.Passengers[1]
	if alice.Name != "Alice" || alice.BoardsAt == nil || *alice.BoardsAt != "Lyon" || *alice.AlightsAt != "Paris" || alice.Balance.Amount != "5.50" {
		t.Errorf("a named passenger: %+v", alice)
	}
	if bob.BoardsAt != nil || bob.AlightsAt != nil {
		t.Errorf("an unlabelled or out-of-range stop is null: %+v", bob)
	}
}

func TestIntegrationCarpoolWithoutTripsIsEmptyNotNull(t *testing.T) {
	out := buildIntegrationCarpool("v1", "CHF", nil, models.CarpoolSummary{}, 10)
	if out.Trips == nil || out.Truncated || out.Summary.NetCostPerKm != nil || out.Summary.CoverageRatePct != nil {
		t.Errorf("no carpool: %+v", out)
	}
}
