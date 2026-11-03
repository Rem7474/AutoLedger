package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/models"
)

// tripFixture records four drives on the account's vehicle: two to Lyon (one with a toll), one stored as coordinates
// and one without distance.
func (f *mcpFixture) tripFixture(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	day := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	km := func(v float64) *float64 { return &v }
	drives := []*models.Drive{
		{StartTime: day, EndTime: day.Add(4 * time.Hour), DistanceKm: km(465), DurationMin: 240, StartAddress: strPtr("12 rue de la Paix, Paris"), EndAddress: strPtr("5 place Bellecour, Lyon"), Tags: []string{"work"}},
		{StartTime: day.AddDate(0, 0, 3), EndTime: day.AddDate(0, 0, 3).Add(4 * time.Hour), DistanceKm: km(465), DurationMin: 240, StartAddress: strPtr("5 place Bellecour, Lyon"), EndAddress: strPtr("12 rue de la Paix, Paris")},
		{StartTime: day.AddDate(0, 0, 10), EndTime: day.AddDate(0, 0, 10).Add(time.Hour), DistanceKm: km(30), DurationMin: 60, StartAddress: strPtr("48.856614, 2.352222"), EndAddress: strPtr("Gare du Nord, Paris")},
		{StartTime: day.AddDate(0, 0, 20), EndTime: day.AddDate(0, 0, 20).Add(time.Hour), DurationMin: 60, StartAddress: strPtr("  "), EndAddress: nil},
	}
	for _, d := range drives {
		d.VehicleID = f.mine.ID
		if err := f.repo.CreateManualDrive(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	toll := &models.DriveExpense{VehicleID: f.mine.ID, DriveID: &drives[0].ID, Type: "TOLL", Amount: 6250, Currency: "GBP", Date: day}
	if err := f.repo.SaveDriveExpense(ctx, toll, nil, ""); err != nil {
		t.Fatal(err)
	}
}

func (f *mcpFixture) trips(t *testing.T, session *mcp.ClientSession, args map[string]any) (IntegrationTrips, *mcp.CallToolResult) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_vehicle_trips", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var out IntegrationTrips
	if !res.IsError {
		raw, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
	}
	return out, res
}

func TestMCPTripsCarryAddressesAndTheirDriveIDButNoPosition(t *testing.T) {
	f := newMCPFixture(t)
	f.tripFixture(t)
	session := f.connect(t, f.readTok)

	all, _ := f.trips(t, session, map[string]any{"vehicle_id": f.mine.ID})
	if all.Matching != 4 || len(all.Trips) != 4 || all.Truncated {
		t.Fatalf("trips: %+v", all)
	}
	raw, _ := json.Marshal(all)
	body := string(raw)
	for _, leak := range []string{"48.856614", "2.352222", f.vin, f.theirs.ID} {
		if strings.Contains(body, leak) {
			t.Errorf("the trips leaked %q: %s", leak, body)
		}
	}
	for _, d := range []string{`"id"`, `"latitude"`} {
		if strings.Contains(body, d) {
			t.Errorf("a trip carries no position and no other identifier than its drive_id, found %s", d)
		}
	}
	for _, trip := range all.Trips {
		if !vehicleIDPattern.MatchString(trip.DriveID) || trip.CarpoolIDs == nil {
			t.Errorf("a trip names its drive, which create_carpool_from_drives takes: %+v", trip)
		}
	}

	// Most recent first: the drive without any address or distance, then the one stored as coordinates.
	last, coords := all.Trips[0], all.Trips[1]
	if last.From != nil || last.To != nil || last.Distance != nil {
		t.Errorf("a blank address and a missing distance are unknown, not empty or zero: %+v", last)
	}
	if coords.From != nil || coords.To == nil || *coords.To != "Gare du Nord, Paris" {
		t.Errorf("an address stored as coordinates is not returned: %+v", coords)
	}
	lyon := all.Trips[3]
	if lyon.To == nil || *lyon.To != "5 place Bellecour, Lyon" || lyon.Distance == nil || lyon.Distance.Value != 465 || lyon.Distance.Unit != "km" {
		t.Errorf("first trip: %+v", lyon)
	}
	if lyon.TravelExpenses == nil || lyon.TravelExpenses.Amount != "62.50" || lyon.TravelExpenses.Currency != "GBP" {
		t.Errorf("the toll is in the vehicle's currency: %+v", lyon.TravelExpenses)
	}
	if all.Trips[2].TravelExpenses != nil {
		t.Errorf("no expense is null, not zero: %+v", all.Trips[2].TravelExpenses)
	}
}

func TestMCPTripsFilterByAddressPeriodAndLimit(t *testing.T) {
	f := newMCPFixture(t)
	f.tripFixture(t)
	session := f.connect(t, f.readTok)

	lyon, _ := f.trips(t, session, map[string]any{"vehicle_id": f.mine.ID, "search": "lyon"})
	if lyon.Matching != 2 || len(lyon.Trips) != 2 {
		t.Errorf("a trip to or from Lyon: %+v", lyon)
	}
	window, _ := f.trips(t, session, map[string]any{"vehicle_id": f.mine.ID, "from": "2026-03-13", "to": "2026-03-13"})
	if window.Matching != 1 || window.Trips[0].To == nil || *window.Trips[0].To != "12 rue de la Paix, Paris" {
		t.Errorf("a single day includes the whole day: %+v", window)
	}
	one, _ := f.trips(t, session, map[string]any{"vehicle_id": f.mine.ID, "limit": 1})
	if len(one.Trips) != 1 || one.Matching != 4 || !one.Truncated {
		t.Errorf("limit: %+v", one)
	}
	if _, res := f.trips(t, session, map[string]any{"vehicle_id": f.mine.ID, "from": "not a date"}); !res.IsError {
		t.Error("a bad date must be an error the assistant can read")
	}
}

func TestMCPTripsOfAnotherAccountAreNotFound(t *testing.T) {
	f := newMCPFixture(t)
	f.tripFixture(t)
	if _, res := f.trips(t, f.connect(t, f.readTok), map[string]any{"vehicle_id": f.theirs.ID}); !res.IsError {
		t.Error("another account's vehicle must be an error")
	}
	if got, _ := f.trips(t, f.connect(t, f.otherTok), map[string]any{"vehicle_id": f.theirs.ID}); got.Matching != 0 {
		t.Errorf("the other account has no trips: %+v", got)
	}
}
