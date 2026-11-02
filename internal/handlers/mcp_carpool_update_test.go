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
)

// recordedCarpool creates a two-drive carpool with Alice (who paid 12.50) and Bob (who boarded at the second stop).
func recordedCarpool(t *testing.T, a *carpoolAPI, h *IntegrationReadHandler) IntegrationCarpoolTrip {
	t.Helper()
	first := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	second := manualDrive(t, a, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC), "Valence, France", "Nice, France", 200)
	out, err := h.CreateCarpoolFromDrives(context.Background(), a.userID, a.vid, mcpCreateCarpoolInput{
		DriveIDs: []string{first.ID, second.ID}, Notes: "week-end",
		Passengers: []mcpCarpoolPassengerInput{{Name: "Alice", AmountPaid: "12.50"}, {Name: "Bob", AmountPaid: "0", BoardsAtStop: ptr(1)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Carpool
}

func TestUpdateCarpoolChangesPassengersAndKeepsTheRest(t *testing.T) {
	a := newCarpoolAPI(t, "mcpupdate")
	ctx := context.Background()
	h := writeHandler(a)
	before := recordedCarpool(t, a, h)
	stored, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil {
		t.Fatal(err)
	}

	// A dry run shows the result and stores nothing
	preview, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{
		CarpoolID: before.ID, DryRun: true, UpdatePassengers: []mcpPassengerChange{{Name: "bob", AmountPaid: ptr("8")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Saved || preview.Carpool.ID != before.ID || preview.Carpool.Revenue.Amount != "20.50" {
		t.Fatalf("a dry run shows the carpool as it would be: %+v", preview)
	}
	if again, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid); err != nil || again.TotalRevenue != stored.TotalRevenue {
		t.Fatalf("a dry run changed the carpool: %+v %v", again, err)
	}

	out, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{
		CarpoolID: before.ID, Title: ptr("  Week-end Nice "), Notes: ptr(""),
		UpdatePassengers: []mcpPassengerChange{
			{Name: " BOB ", AmountPaid: ptr("8"), NewName: ptr("Robert"), Seats: ptr(2), Notes: ptr("front seat")},
			{Name: "Alice", AlightsAtStop: ptr(1)},
		},
		AddPassengers: []mcpCarpoolPassengerInput{{Name: "Chloé", AmountPaid: "5"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := out.Carpool
	if !out.Saved || got.ID != before.ID || got.Title != "Week-end Nice" || got.Notes != nil || got.Distance.Value != 300 || got.Revenue.Amount != "25.50" {
		t.Fatalf("saved carpool: %+v", out)
	}
	if len(got.Passengers) != 3 {
		t.Fatalf("passengers: %+v", got.Passengers)
	}
	alice, bob, chloe := got.Passengers[0], got.Passengers[1], got.Passengers[2]
	if alice.Name != "Alice" || alice.Paid.Amount != "12.50" || alice.AlightsAt == nil || *alice.AlightsAt != "Valence" {
		t.Errorf("only the fields given change: %+v", alice)
	}
	if bob.Name != "Robert" || bob.Paid.Amount != "8.00" || bob.Seats != 2 || bob.BoardsAt == nil || *bob.BoardsAt != "Valence" || bob.Notes == nil || *bob.Notes != "front seat" {
		t.Errorf("changed passenger: %+v", bob)
	}
	if chloe.Name != "Chloé" || chloe.Paid.Amount != "5.00" {
		t.Errorf("added passenger: %+v", chloe)
	}

	// The legs, the date and the trip group stay what they were
	after, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil || len(after.Legs) != 2 || !after.Date.Equal(stored.Date) || after.TripGroupID == nil || *after.TripGroupID != *stored.TripGroupID {
		t.Fatalf("legs, date and trip group are kept: %+v %v", after, err)
	}
	for i, l := range after.Legs {
		if l.DriveID == nil || stored.Legs[i].DriveID == nil || *l.DriveID != *stored.Legs[i].DriveID {
			t.Errorf("leg %d changed drive", i)
		}
	}

	// Taking a passenger out takes what they paid out of the revenue
	removed, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, RemovePassengers: []string{"alice"}})
	if err != nil || len(removed.Carpool.Passengers) != 2 || removed.Carpool.Revenue.Amount != "13.00" {
		t.Fatalf("removal: %+v %v", removed, err)
	}
	if trips, err := a.repo.ListCarpoolTrips(ctx, a.vid); err != nil || len(trips) != 1 {
		t.Fatalf("an update never creates or deletes a carpool: %v %v", trips, err)
	}
}

func TestUpdateCarpoolKeepsAFrozenCarpoolsAmounts(t *testing.T) {
	a := newCarpoolAPI(t, "mcpupdate-frozen")
	ctx := context.Background()
	h := writeHandler(a)
	before := recordedCarpool(t, a, h)
	stored, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	stored.CostsFrozenAt = &now
	if err := a.repo.UpdateCarpoolTrip(ctx, &stored.CarpoolTrip, stored.Legs, stored.Passengers); err != nil {
		t.Fatal(err)
	}
	frozen, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil || frozen.CostsFrozenAt == nil {
		t.Fatalf("setup: %+v %v", frozen, err)
	}

	if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, Title: ptr("Renamed")}); err != nil {
		t.Fatal(err)
	}
	after, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil || after.CostsFrozenAt == nil || after.Title != "Renamed" || after.TotalCost != frozen.TotalCost {
		t.Fatalf("a frozen carpool stays frozen with the same costs: %+v %v", after, err)
	}
}

func TestUpdateCarpoolRefusesWhatItCannotChange(t *testing.T) {
	a := newCarpoolAPI(t, "mcpupdate-refuse")
	ctx := context.Background()
	h := writeHandler(a)
	c := recordedCarpool(t, a, h)
	other := "00000000-0000-4000-8000-000000000001"

	cases := map[string]struct {
		user string
		in   mcpUpdateCarpoolInput
		want string
	}{
		"nothing to change":     {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID}, "nothing to change"},
		"not an id":             {a.userID, mcpUpdateCarpoolInput{CarpoolID: "Lyon", Title: ptr("x")}, "not a carpool id"},
		"unknown carpool":       {a.userID, mcpUpdateCarpoolInput{CarpoolID: other, Title: ptr("x")}, "carpool not found"},
		"unknown passenger":     {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Zoe", Seats: ptr(2)}}}, `no passenger is called "Zoe": the carpool has "Alice", "Bob"`},
		"remove unknown":        {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, RemovePassengers: []string{"Zoe"}}, "no passenger is called"},
		"add a name twice":      {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, AddPassengers: []mcpCarpoolPassengerInput{{Name: "alice"}}}, "already in the carpool"},
		"negative amount":       {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Bob", AmountPaid: ptr("-1")}}}, "amount_paid of Bob"},
		"not an amount":         {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Bob", AmountPaid: ptr("ten")}}}, "amount_paid of Bob"},
		"empty new name":        {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Bob", NewName: ptr(" ")}}}, "new_name cannot be empty"},
		"stops out of the trip": {a.userID, mcpUpdateCarpoolInput{CarpoolID: c.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Bob", AlightsAtStop: ptr(5)}}}, "drop-off stop"},
		"read-only member":      {a.viewerID, mcpUpdateCarpoolInput{CarpoolID: c.ID, Title: ptr("x")}, "read-only"},
	}
	for name, tc := range cases {
		if _, err := h.UpdateCarpool(ctx, tc.user, a.vid, tc.in); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
	after, err := a.repo.GetCarpoolTrip(ctx, c.ID, a.vid)
	if err != nil || after.Title != c.Title || len(after.Passengers) != 2 || after.TotalRevenue != 1250 {
		t.Errorf("a refused call changed the carpool: %+v %v", after, err)
	}
}

func TestUpdateCarpoolTellsHomonymsApart(t *testing.T) {
	a := newCarpoolAPI(t, "mcpupdate-homonyms")
	ctx := context.Background()
	h := writeHandler(a)
	d := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon", "Paris", 100)
	out, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{
		DriveIDs: []string{d.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "Sam", AmountPaid: "1"}, {Name: "sam", AmountPaid: "2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: out.Carpool.ID, UpdatePassengers: []mcpPassengerChange{{Name: "Sam", AmountPaid: ptr("9")}}})
	if err == nil || !strings.Contains(err.Error(), "cannot be told apart") {
		t.Fatalf("a name shared by two passengers is never guessed: %v", err)
	}
}

func TestMCPUpdateCarpoolNeedsAConnectionAllowedToWrite(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	km := 80.0
	from, to := "Lyon", "Annecy"
	start := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	d := &models.Drive{VehicleID: f.mine.ID, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: &km, DurationMin: 60, StartAddress: &from, EndAddress: &to}
	if err := f.repo.CreateManualDrive(ctx, d); err != nil {
		t.Fatal(err)
	}
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token, name string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	created := call(writeTok, "create_carpool_from_drives", map[string]any{"vehicle_id": f.mine.ID, "drive_ids": []string{d.ID}, "passengers": []map[string]any{{"name": "Alice", "amount_paid": "9"}}})
	raw, _ := json.Marshal(created.StructuredContent)
	var made IntegrationCarpoolCreated
	if err := json.Unmarshal(raw, &made); err != nil || made.Carpool.ID == "" {
		t.Fatalf("setup: %s %v", raw, err)
	}
	args := map[string]any{"vehicle_id": f.mine.ID, "carpool_id": made.Carpool.ID, "update_passengers": []map[string]any{{"name": "Alice", "amount_paid": "11"}}}

	if res := call(f.readTok, "update_carpool", args); !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, "can only read") {
		t.Fatalf("a read connection must not change a carpool: %+v", res)
	}
	if stored, err := f.repo.GetCarpoolTrip(ctx, made.Carpool.ID, f.mine.ID); err != nil || stored.TotalRevenue != 900 {
		t.Fatalf("a refused call changed the carpool: %+v %v", stored, err)
	}

	res := call(writeTok, "update_carpool", args)
	if res.IsError {
		t.Fatalf("a write connection changes: %+v", res.Content)
	}
	raw, _ = json.Marshal(res.StructuredContent)
	var out IntegrationCarpoolUpdated
	if err := json.Unmarshal(raw, &out); err != nil || !out.Saved || out.Carpool.Revenue.Amount != "11.00" {
		t.Fatalf("saved: %s %v", raw, err)
	}

	args["vehicle_id"] = f.theirs.ID
	if res := call(writeTok, "update_carpool", args); !res.IsError {
		t.Error("a vehicle of another account must be refused")
	}
}
