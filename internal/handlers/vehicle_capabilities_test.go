package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
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

	// A combustion vehicle enters its drives by hand: it has the trips capability from its creation, with no drive stored.
	ice := decodeObj(t, a.do(a.other, http.StatusCreated, "POST", "/", `{"name":"Clio","currency":"EUR","powertrain":"ICE"}`))
	iceID := ice["id"].(string)
	if got := capabilitiesOf(t, ice); got["refuels"] != true || got["charges"] != false || got["drives"] != true || got["battery_health"] != false {
		t.Fatalf("a combustion vehicle: %v", got)
	}
	got := decodeObj(t, a.do(a.other, http.StatusOK, "GET", "/"+iceID, ""))
	if c := capabilitiesOf(t, got); c["drives"] != true || c["refuels"] != true {
		t.Fatalf("a combustion vehicle read back: %v", c)
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
