package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// manualDrive stores a one-hour drive between two named places on the vehicle.
func manualDrive(t *testing.T, a *carpoolAPI, start time.Time, from, to string, km float64) *models.Drive {
	t.Helper()
	d := &models.Drive{VehicleID: a.vid, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: &km, DurationMin: 60, StartAddress: &from, EndAddress: &to}
	if err := a.repo.CreateManualDrive(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return d
}

func writeHandler(a *carpoolAPI) *IntegrationReadHandler {
	return NewIntegrationReadHandler(a.repo, nil).WithCarpool(services.NewCarpoolService(a.repo.Pool(), a.repo))
}

func TestCreateCarpoolFromDrivesGroupsTheDrivesAndNamesWhoPaid(t *testing.T) {
	a := newCarpoolAPI(t, "mcpwrite")
	ctx := context.Background()
	first := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	second := manualDrive(t, a, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC), "Valence, France", "Nice, France", 200)
	h := writeHandler(a)
	paid := "12.50"
	// Given in reverse order: the legs follow the chronology
	in := mcpCreateCarpoolInput{
		DriveIDs: []string{second.ID, first.ID}, Notes: "  week-end  ",
		Passengers: []mcpCarpoolPassengerInput{{Name: " Alice ", AmountPaid: paid}, {Name: "Bob", AmountPaid: "0", BoardsAtStop: ptr(1)}},
	}

	preview := in
	preview.DryRun = true
	dry, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, preview)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Recorded || dry.Carpool.ID != "" || dry.Carpool.Title != "Lyon → Nice" || len(dry.Carpool.Stops) != 3 {
		t.Fatalf("a dry run shows the carpool and records nothing: %+v", dry)
	}
	if trips, err := a.repo.ListCarpoolTrips(ctx, a.vid); err != nil || len(trips) != 0 {
		t.Fatalf("a dry run stored a carpool: %v %v", trips, err)
	}

	out, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, in)
	if err != nil {
		t.Fatal(err)
	}
	got := out.Carpool
	if !out.Recorded || got.ID == "" || got.Title != "Lyon → Nice" || got.Distance.Value != 300 || got.Revenue.Amount != "12.50" {
		t.Fatalf("recorded carpool: %+v", out)
	}
	if got.Notes == nil || *got.Notes != "week-end" || len(got.Passengers) != 2 {
		t.Fatalf("notes and passengers: %+v", got)
	}
	alice, bob := got.Passengers[0], got.Passengers[1]
	if alice.Name != "Alice" || alice.Paid.Amount != "12.50" || alice.BoardsAt == nil || *alice.BoardsAt != "Lyon" || *alice.AlightsAt != "Nice" {
		t.Errorf("a passenger rides the whole trip by default: %+v", alice)
	}
	if bob.BoardsAt == nil || *bob.BoardsAt != "Valence" || bob.Paid.Amount != "0.00" {
		t.Errorf("a passenger can board at a later stop: %+v", bob)
	}

	// The same figures as the carpool the editor would have saved for these drives
	stored, err := a.repo.GetCarpoolTrip(ctx, got.ID, a.vid)
	if err != nil || len(stored.Legs) != 2 || *stored.Legs[0].DriveID != first.ID || *stored.Legs[1].DriveID != second.ID {
		t.Fatalf("legs in chronological order: %+v %v", stored, err)
	}
	if stored.TripGroupID == nil {
		t.Error("a carpool of several drives is a trip group")
	}

	if _, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{DriveIDs: []string{first.ID}}); err == nil || !strings.Contains(err.Error(), "already in the carpool") {
		t.Errorf("a drive is shared once: %v", err)
	}
	trips, err := h.Trips(ctx, a.userID, a.vid, IntegrationTripsQuery{})
	if err != nil || len(trips.Trips) != 2 || trips.Trips[0].DriveID != second.ID || len(trips.Trips[0].CarpoolIDs) != 1 || trips.Trips[0].CarpoolIDs[0] != got.ID {
		t.Errorf("list_vehicle_trips names the drives and the carpool they are in: %+v %v", trips, err)
	}
}

func TestCreateCarpoolFromDrivesRefusesWhatItCannotRecord(t *testing.T) {
	a := newCarpoolAPI(t, "mcpwrite-refuse")
	ctx := context.Background()
	d := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon", "Paris", 100)
	nameless := &models.Drive{VehicleID: a.vid, StartTime: time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC), DurationMin: 60}
	km := 50.0
	nameless.DistanceKm = &km
	if err := a.repo.CreateManualDrive(ctx, nameless); err != nil {
		t.Fatal(err)
	}
	h := writeHandler(a)
	other := "00000000-0000-4000-8000-000000000001"
	long := make([]mcpCarpoolPassengerInput, 8)
	for i := range long {
		long[i] = mcpCarpoolPassengerInput{Name: "P"}
	}

	cases := map[string]struct {
		user string
		in   mcpCreateCarpoolInput
		want string
	}{
		"no drive":              {a.userID, mcpCreateCarpoolInput{}, "drive_ids is empty"},
		"not an id":             {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{"Lyon"}}, "not a drive id"},
		"twice":                 {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID, d.ID}}, "listed twice"},
		"unknown drive":         {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{other}}, "not a drive of this vehicle"},
		"no address, no title":  {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{nameless.ID}}, "title is required"},
		"nameless passenger":    {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: " "}}}, "needs a name"},
		"negative amount":       {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "A", AmountPaid: "-3"}}}, "amount_paid of A"},
		"not an amount":         {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "A", AmountPaid: "ten"}}}, "amount_paid of A"},
		"too many passengers":   {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}, Passengers: long}, "7 passengers at most"},
		"drop-off before start": {a.userID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "A", BoardsAtStop: ptr(1), AlightsAtStop: ptr(0)}}}, "drop-off stop"},
		"read-only member":      {a.viewerID, mcpCreateCarpoolInput{DriveIDs: []string{d.ID}}, "read-only"},
	}
	for name, c := range cases {
		if _, err := h.CreateCarpoolFromDrives(ctx, c.user, a.vid, c.in); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	if trips, err := a.repo.ListCarpoolTrips(ctx, a.vid); err != nil || len(trips) != 0 {
		t.Errorf("a refused call stored a carpool: %v %v", trips, err)
	}
}

func TestMCPCreateCarpoolNeedsAConnectionAllowedToWrite(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	km := 80.0
	from, to := "Lyon", "Annecy"
	start := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	d := &models.Drive{VehicleID: f.mine.ID, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: &km, DurationMin: 60, StartAddress: &from, EndAddress: &to}
	if err := f.repo.CreateManualDrive(ctx, d); err != nil {
		t.Fatal(err)
	}
	call := func(token string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "create_carpool_from_drives", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	args := map[string]any{"vehicle_id": f.mine.ID, "drive_ids": []string{d.ID}, "passengers": []map[string]any{{"name": "Alice", "amount_paid": "9"}}}

	if res := call(f.readTok, args); !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "can only read") {
		t.Fatalf("a read connection must not write: %+v", res)
	}
	if trips, err := f.repo.ListCarpoolTrips(ctx, f.mine.ID); err != nil || len(trips) != 0 {
		t.Fatalf("a refused call stored a carpool: %v %v", trips, err)
	}

	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	res := call(writeTok, args)
	if res.IsError {
		t.Fatalf("a write connection records: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var out IntegrationCarpoolCreated
	if err := json.Unmarshal(raw, &out); err != nil || !out.Recorded || out.Carpool.ID == "" || out.Carpool.Revenue.Amount != "9.00" {
		t.Fatalf("recorded: %s %v", raw, err)
	}

	theirs := map[string]any{"vehicle_id": f.theirs.ID, "drive_ids": []string{d.ID}}
	if res := call(writeTok, theirs); !res.IsError {
		t.Error("a vehicle of another account must be refused")
	}

	// A drive in a carpool is refused, a dry run included
	if res := call(writeTok, map[string]any{"vehicle_id": f.mine.ID, "drive_ids": []string{d.ID}, "dry_run": true}); !res.IsError {
		t.Error("the drive is now in a carpool")
	}
}
