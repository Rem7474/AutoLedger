package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestHybridShareEndpoint(t *testing.T) {
	a := newVehicleAPI(t, "hybrid-share")
	a.router.Get("/{vehicleId}/hybrid-share", NewHybridShareHandler(a.repo).Get)
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
	// The suggestion is read-only: the vehicle carries nothing new.
	a.do(a.other, 404, "GET", "/"+phev+"/hybrid-share", "")
}
