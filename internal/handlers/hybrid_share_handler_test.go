package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestHybridShareEndpoint(t *testing.T) {
	a := newVehicleAPI(t, "hybrid-share")
	a.router.Get("/{vehicleId}/hybrid-share", NewHybridShareHandler(a.repo).Get)
	a.router.Put("/{vehicleId}/hybrid-share", NewHybridShareHandler(a.repo).Save)
	ctx := context.Background()

	ice := decodeObj(t, a.do(a.ownerID, 201, "POST", "/", `{"name":"Diesel","powertrain":"ICE","currency":"EUR"}`))["id"].(string)
	a.do(a.ownerID, 400, "GET", "/"+ice+"/hybrid-share", "")
	ev := decodeObj(t, a.do(a.ownerID, 201, "POST", "/", `{"name":"Electric","powertrain":"EV","currency":"EUR"}`))["id"].(string)
	a.do(a.ownerID, 400, "GET", "/"+ev+"/hybrid-share", "")

	phev := decodeObj(t, a.do(a.ownerID, 201, "POST", "/", `{"name":"Plug-in","powertrain":"PHEV","currency":"EUR","estimated_kwh_100km":16}`))["id"].(string)
	// Nothing entered yet: no suggestion, and the level says why.
	empty := decodeObj(t, a.do(a.ownerID, 200, "GET", "/"+phev+"/hybrid-share", ""))
	if empty["suggested_electric_share"] != nil || empty["quality"].(map[string]any)["level"] != "insufficient" {
		t.Fatalf("empty history: %v", empty)
	}

	// Six periods of 300 km, 40 % electric: 19.2 kWh and 10.8 L each.
	day := time.Now().UTC().AddDate(0, 0, -70)
	odo := 5000.0
	fill := func(at time.Time, odometer, liters float64) {
		l, o := liters, odometer
		if err := a.repo.CreateFuelLog(ctx, &models.FuelLog{VehicleID: phev, Date: at, Odometer: &o, Liters: &l, Amount: money.FromFloat(liters * 1.8), IsFullTank: true}); err != nil {
			t.Fatal(err)
		}
	}
	fill(day, odo, 40)
	for i := 0; i < 6; i++ {
		used, o := 19.2, odo+150 // a manual charge carries what the owner read: the plug side
		if err := a.repo.CreateManualCharge(ctx, &models.ChargeLog{VehicleID: phev, Date: day.AddDate(0, 0, 4), KwhAdded: used, Odometer: &o, Currency: "EUR"}); err != nil {
			t.Fatal(err)
		}
		odo += 300
		day = day.AddDate(0, 0, 10)
		fill(day, odo, 10.8)
	}
	if _, err := a.repo.AddVehicleMember(ctx, phev, "viewer-hybrid-share@example.com", models.RoleViewer, nil); err != nil {
		t.Fatal(err)
	}
	got := decodeObj(t, a.do(a.viewer, 200, "GET", "/"+phev+"/hybrid-share", ""))
	if share, _ := got["suggested_electric_share"].(float64); share < 0.399 || share > 0.401 {
		t.Fatalf("share: %v", got)
	}
	q := got["quality"].(map[string]any)
	if q["level"] != "good" || q["segments_used"] != float64(6) || got["method"] != "reference" {
		t.Fatalf("quality: %v", got)
	}
	// The suggestion is read-only: the vehicle carries nothing accepted.
	if acc := got["accepted"].(map[string]any); acc["electric_share"] != nil || acc["fuel_l_100km"] != nil {
		t.Fatalf("accepted: %v", acc)
	}
	a.do(a.other, 404, "GET", "/"+phev+"/hybrid-share", "")

	// Only an editor decides.
	a.do(a.viewer, 403, "PUT", "/"+phev+"/hybrid-share", `{"accept_suggestion":true}`)
	a.do(a.other, 404, "PUT", "/"+phev+"/hybrid-share", `{"accept_suggestion":true}`)
	// Accepting stores the suggestion, with the fuel consumption of the fuel part.
	saved := decodeObj(t, a.do(a.ownerID, 200, "PUT", "/"+phev+"/hybrid-share", `{"accept_suggestion":true,"battery_usable_kwh":null}`))
	if share, _ := saved["electric_share"].(float64); share < 0.399 || share > 0.401 {
		t.Fatalf("saved: %v", saved)
	}
	if fuelL, _ := saved["fuel_l_100km"].(float64); fuelL < 5.9 || fuelL > 6.1 {
		t.Fatalf("saved fuel: %v", saved)
	}
	again := decodeObj(t, a.do(a.ownerID, 200, "GET", "/"+phev+"/hybrid-share", ""))
	if acc := again["accepted"].(map[string]any); acc["electric_share"] != saved["electric_share"] {
		t.Fatalf("accepted after save: %v", acc)
	}
	// Typed values replace the accepted ones; null clears one; a consumption without a share is dropped.
	typed := decodeObj(t, a.do(a.ownerID, 200, "PUT", "/"+phev+"/hybrid-share", `{"electric_share":0.55,"fuel_l_100km":5.5,"battery_usable_kwh":12}`))
	if typed["electric_share"] != 0.55 || typed["fuel_l_100km"] != 5.5 || typed["battery_usable_kwh"] != float64(12) {
		t.Fatalf("typed: %v", typed)
	}
	cleared := decodeObj(t, a.do(a.ownerID, 200, "PUT", "/"+phev+"/hybrid-share", `{"fuel_l_100km":5.5,"battery_usable_kwh":12}`))
	if cleared["electric_share"] != nil || cleared["fuel_l_100km"] != nil || cleared["battery_usable_kwh"] != float64(12) {
		t.Fatalf("cleared: %v", cleared)
	}
	for _, bad := range []string{`{"electric_share":1.2}`, `{"electric_share":-0.1}`, `{"electric_share":0.5,"fuel_l_100km":0}`, `{"electric_share":0.5,"fuel_l_100km":80}`, `{"battery_usable_kwh":0}`, `{"battery_usable_kwh":900}`, `{`} {
		a.do(a.ownerID, 400, "PUT", "/"+phev+"/hybrid-share", bad)
	}
	// Nothing to suggest, nothing to accept.
	empty2 := decodeObj(t, a.do(a.ownerID, 201, "POST", "/", `{"name":"Plug-in 2","powertrain":"REEV","currency":"EUR"}`))["id"].(string)
	a.do(a.ownerID, 409, "PUT", "/"+empty2+"/hybrid-share", `{"accept_suggestion":true}`)
	a.do(a.ownerID, 400, "PUT", "/"+ice+"/hybrid-share", `{"electric_share":0.5}`)
}

func TestBuildChargeBatteryLevels(t *testing.T) {
	i := func(v int) *int { return &v }
	ok := []struct{ start, end *int }{{nil, nil}, {i(20), i(80)}, {nil, i(90)}, {i(0), i(100)}, {i(30), nil}}
	for _, c := range ok {
		if _, err := buildCharge("v", "EUR", &SaveChargeRequest{Date: "2026-10-05", KwhAdded: 10, StartBatteryLevel: c.start, EndBatteryLevel: c.end}); err != nil {
			t.Fatalf("%v %v: %v", c.start, c.end, err)
		}
	}
	bad := map[string]struct{ start, end *int }{
		"battery_level_invalid":  {i(-1), i(50)},
		"battery_level_invalid ": {i(20), i(101)},
		"battery_levels_order":   {i(80), i(80)},
		"battery_levels_order ":  {i(80), i(20)},
	}
	for want, c := range bad {
		_, err := buildCharge("v", "EUR", &SaveChargeRequest{Date: "2026-10-05", KwhAdded: 10, StartBatteryLevel: c.start, EndBatteryLevel: c.end})
		if errorCode(err) != "charge."+strings.TrimSpace(want) {
			t.Fatalf("%s: %v", want, err)
		}
	}
	c, _ := buildCharge("v", "EUR", &SaveChargeRequest{Date: "2026-10-05", KwhAdded: 10, StartBatteryLevel: i(20), EndBatteryLevel: i(80)})
	if *c.StartBatteryLevel != 20 || *c.EndBatteryLevel != 80 {
		t.Fatalf("%+v", c)
	}
}

func TestChargeBatteryLevelsRoundTrip(t *testing.T) {
	a := newVehicleAPI(t, "charge-levels")
	ctx := context.Background()
	start, end, odo := 25, 90, 12000.0
	c := &models.ChargeLog{VehicleID: a.vid, Date: time.Now().UTC().Add(-time.Hour), KwhAdded: 20, Currency: "EUR", Odometer: &odo, StartBatteryLevel: &start, EndBatteryLevel: &end}
	if err := a.repo.CreateManualCharge(ctx, c); err != nil {
		t.Fatal(err)
	}
	list, _, err := a.repo.ListCharges(ctx, a.vid, false, 10, 0)
	if err != nil || len(list) != 1 || list[0].StartBatteryLevel == nil || *list[0].StartBatteryLevel != 25 || *list[0].EndBatteryLevel != 90 {
		t.Fatalf("read back: %+v %v", list, err)
	}
	// An edit replaces them as sent.
	newStart := 30
	c.StartBatteryLevel = &newStart
	if err := a.repo.UpdateCharge(ctx, c); err != nil || *c.StartBatteryLevel != 30 || *c.EndBatteryLevel != 90 {
		t.Fatalf("update: %+v %v", c, err)
	}
	c.StartBatteryLevel, c.EndBatteryLevel = nil, nil
	if err := a.repo.UpdateCharge(ctx, c); err != nil || c.StartBatteryLevel != nil || c.EndBatteryLevel != nil {
		t.Fatalf("cleared: %+v %v", c, err)
	}
}
