package handlers

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestUpdateCarpoolLinksADriveAndMovesThePassengersToTheNewEnd(t *testing.T) {
	a := newCarpoolAPI(t, "mcplink")
	ctx := context.Background()
	h := writeHandler(a)
	before := recordedCarpool(t, a, h)
	third := manualDrive(t, a, time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC), "Nice, France", "Cannes, France", 50)

	preview, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, AddDrives: []string{third.ID}, DryRun: true})
	if err != nil || preview.Saved || preview.Carpool.Distance.Value != 350 {
		t.Fatalf("a dry run shows the longer carpool: %+v %v", preview, err)
	}
	if stored, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid); err != nil || len(stored.Legs) != 2 {
		t.Fatalf("a dry run linked the drive: %+v %v", stored, err)
	}

	out, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, AddDrives: []string{third.ID}})
	if err != nil || !out.Saved || out.Carpool.Distance.Value != 350 || out.Carpool.Revenue.Amount != "12.50" {
		t.Fatalf("link: %+v %v", out, err)
	}
	stored, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil || len(stored.Legs) != 3 || stored.Legs[2].DriveID == nil || *stored.Legs[2].DriveID != third.ID {
		t.Fatalf("the drive is the last leg: %+v %v", stored, err)
	}
	// Alice and Bob rode to the end of the trip: they ride to the end of the longer one
	alice, bob := stored.Passengers[0], stored.Passengers[1]
	if alice.BoardStopIndex != 0 || alice.AlightStopIndex != 3 || bob.BoardStopIndex != 1 || bob.AlightStopIndex != 3 {
		t.Errorf("stops: alice %d-%d, bob %d-%d", alice.BoardStopIndex, alice.AlightStopIndex, bob.BoardStopIndex, bob.AlightStopIndex)
	}
	if alice.Destination == nil || *alice.Destination != "Cannes" {
		t.Errorf("the labels follow the stops: %+v", alice.Destination)
	}

	// A drive is in one carpool at a time, and a leg is not linked twice
	other := manualDrive(t, a, time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC), "Lyon, France", "Paris, France", 460)
	if _, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{DriveIDs: []string{other.ID}, Title: "Elsewhere"}); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		ids  []string
		want string
	}{
		"in another carpool": {[]string{other.ID}, "is already in the carpool"},
		"already a leg":      {[]string{third.ID}, "already a leg of this carpool"},
		"not a drive":        {[]string{"00000000-0000-4000-8000-000000000001"}, "not a drive of this vehicle"},
	} {
		if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: out.Carpool.Version, AddDrives: tc.ids}); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
}

func TestUpdateCarpoolUnlinksADriveAndKeepsTheDrive(t *testing.T) {
	a := newCarpoolAPI(t, "mcpunlink")
	ctx := context.Background()
	h := writeHandler(a)
	before := recordedCarpool(t, a, h)
	stored, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil {
		t.Fatal(err)
	}
	first, second := *stored.Legs[0].DriveID, *stored.Legs[1].DriveID

	// Bob boarded at the second stop and rode only the second drive: that drive cannot leave while he is in
	if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, RemoveDrives: []string{second}}); err == nil || !strings.Contains(err.Error(), "Bob rode only on the drive taken out") {
		t.Fatalf("a passenger left with no drive: %v", err)
	}
	if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, RemoveDrives: []string{first, second}}); err == nil || !strings.Contains(err.Error(), "at least one drive") {
		t.Fatalf("a carpool keeps a drive: %v", err)
	}
	if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, RemoveDrives: []string{"00000000-0000-4000-8000-000000000001"}}); err == nil || !strings.Contains(err.Error(), "not a leg of this carpool") {
		t.Fatalf("a drive that is not a leg: %v", err)
	}
	if _, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, AddDrives: []string{first}, RemoveDrives: []string{first}}); err == nil || !strings.Contains(err.Error(), "both in add_drives and remove_drives") {
		t.Fatalf("add and remove together: %v", err)
	}

	out, err := h.UpdateCarpool(ctx, a.userID, a.vid, mcpUpdateCarpoolInput{CarpoolID: before.ID, ExpectedVersion: before.Version, RemoveDrives: []string{first}})
	if err != nil || !out.Saved || out.Carpool.Distance.Value != 200 || out.Carpool.Revenue.Amount != "12.50" {
		t.Fatalf("unlink: %+v %v", out, err)
	}
	after, err := a.repo.GetCarpoolTrip(ctx, before.ID, a.vid)
	if err != nil || len(after.Legs) != 1 || *after.Legs[0].DriveID != second {
		t.Fatalf("the other drive is the only leg: %+v %v", after, err)
	}
	if after.DriveID == nil || *after.DriveID != second {
		t.Errorf("a carpool of one drive points at it: %v", after.DriveID)
	}
	// Both passengers now ride the one leg that is left
	for _, p := range after.Passengers {
		if p.BoardStopIndex != 0 || p.AlightStopIndex != 1 {
			t.Errorf("%s rides %d-%d", p.PassengerName, p.BoardStopIndex, p.AlightStopIndex)
		}
	}
	// The drive is untouched and can join another carpool
	if d, err := a.repo.GetDriveByID(ctx, first, a.vid); err != nil || d == nil {
		t.Fatalf("the unlinked drive is kept: %v", err)
	}
	if _, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{DriveIDs: []string{first}, Title: "Alone"}); err != nil {
		t.Fatalf("the unlinked drive is free: %v", err)
	}
}
