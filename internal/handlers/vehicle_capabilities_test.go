package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func capabilitiesOf(t *testing.T, vehicle map[string]any) map[string]any {
	t.Helper()
	caps, ok := vehicle["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("the vehicle carries no capabilities: %v", vehicle)
	}
	return caps
}

func TestVehicleCapabilitiesAreReturnedByCreateUpdateGetAndList(t *testing.T) {
	a := newVehicleAPI(t, "caps")

	created := decodeObj(t, a.do(a.other, http.StatusCreated, "POST", "/", `{"name":"Zoe","currency":"EUR","powertrain":"EV","teslamate_api_url":"http://tm.local"}`))
	caps := capabilitiesOf(t, created)
	if caps["charges"] != true || caps["sync"] != true || caps["odometer_auto"] != true || caps["write"] != true || caps["refuels"] != false {
		t.Fatalf("a connected electric vehicle created by its owner: %v", caps)
	}

	updated := decodeObj(t, a.do(a.other, http.StatusOK, "PUT", "/"+created["id"].(string), `{"name":"Zoe 2","teslamate_api_url":"http://tm.local"}`))
	if got := capabilitiesOf(t, updated); got["sync"] != true || got["write"] != true {
		t.Fatalf("an update returns the capabilities of the saved vehicle: %v", got)
	}
	disconnected := decodeObj(t, a.do(a.other, http.StatusOK, "PUT", "/"+created["id"].(string), `{"name":"Zoe 2"}`))
	if got := capabilitiesOf(t, disconnected); got["sync"] != false || got["odometer_auto"] != false {
		t.Fatalf("removing the connection removes the capability: %v", got)
	}

	ice := decodeObj(t, a.do(a.other, http.StatusCreated, "POST", "/", `{"name":"Clio","currency":"EUR","powertrain":"ICE"}`))
	iceID := ice["id"].(string)
	if got := capabilitiesOf(t, ice); got["refuels"] != true || got["charges"] != false || got["drives"] != false || got["battery_health"] != false {
		t.Fatalf("a combustion vehicle: %v", got)
	}

	// A drive written by a push source gives a combustion vehicle the trips capability.
	start := time.Now().Add(-time.Hour)
	dist := 12.0
	externalID := "obd-1"
	drive := &models.Drive{VehicleID: iceID, Origin: "WEBHOOK", ExternalID: &externalID, StartTime: start, EndTime: start.Add(20 * time.Minute), DistanceKm: &dist, Tags: []string{}, IsManual: true}
	if err := a.repo.CreateManualDrive(context.Background(), drive); err != nil {
		t.Fatal(err)
	}
	got := decodeObj(t, a.do(a.other, http.StatusOK, "GET", "/"+iceID, ""))
	if c := capabilitiesOf(t, got); c["drives"] != true || c["refuels"] != true {
		t.Fatalf("a combustion vehicle with a pushed drive: %v", c)
	}

	var list []map[string]any
	if err := json.Unmarshal(a.do(a.other, http.StatusOK, "GET", "/", ""), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two vehicles, got %d", len(list))
	}
	for _, v := range list {
		c := capabilitiesOf(t, v)
		if v["id"] == iceID && c["drives"] != true {
			t.Errorf("the list gives the combustion vehicle its trips: %v", c)
		}
	}
}

func TestVehicleCapabilitiesFollowTheRoleOfTheMember(t *testing.T) {
	a := newVehicleAPI(t, "caps-role")
	for user, write := range map[string]bool{a.ownerID: true, a.editor: true, a.viewer: false} {
		got := capabilitiesOf(t, decodeObj(t, a.do(user, http.StatusOK, "GET", "/"+a.vid, "")))
		if got["write"] != write || got["charges"] != true || got["drives"] != true {
			t.Errorf("member %s: %v", user, got)
		}
	}
}
