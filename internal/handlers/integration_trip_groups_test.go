package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func TestIntegrationTripGroupsSummariseEachJourneyAndCapTheList(t *testing.T) {
	start, end := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC), time.Date(2026, 7, 9, 18, 0, 0, 0, time.UTC)
	note := "ferry booked"
	groups := []models.TripGroup{
		{Name: "Holidays", DriveIDs: []string{"a", "b", "c"}, DistanceKm: 1840.5, StartTime: &start, EndTime: &end, TollsTotal: 12050, ExpenseCount: 2, CarpoolCount: 1, Notes: &note},
		{Name: "Empty"},
	}

	out := buildIntegrationTripGroups("v1", "CHF", groups, 1)

	if out.Matching != 2 || !out.Truncated || len(out.Groups) != 1 {
		t.Fatalf("the cap: %+v", out)
	}
	g := out.Groups[0]
	if g.Name != "Holidays" || g.Drives != 3 || g.Distance.Value != 1840.5 || g.TravelExpenses.Amount != "120.50" || g.GroupExpenses != 2 || g.CarpoolTrips != 1 ||
		g.Start == nil || *g.Start != "2026-07-01T08:00:00Z" || g.Notes == nil {
		t.Errorf("a trip group: %+v", g)
	}
	empty := buildIntegrationTripGroups("v1", "CHF", groups[1:], 5)
	if empty.Truncated || empty.Groups[0].Start != nil || empty.Groups[0].End != nil || empty.Groups[0].Notes != nil {
		t.Errorf("a group without drive has no period: %+v", empty.Groups[0])
	}
	if none := buildIntegrationTripGroups("v1", "CHF", nil, 5); none.Groups == nil {
		t.Error("no group is an empty list")
	}
}

func TestIntegrationTripNamesItsDriver(t *testing.T) {
	name := " Camille "
	trip := integrationTrip(models.Drive{DriverName: &name}, "CHF", nil)
	if trip.Driver == nil || *trip.Driver != "Camille" {
		t.Errorf("driver: %v", trip.Driver)
	}
	if none := integrationTrip(models.Drive{}, "CHF", nil); none.Driver != nil {
		t.Errorf("no driver is null: %v", none.Driver)
	}
}
