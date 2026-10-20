package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func (a *carpoolAPI) driveWithToll(distanceKm float64, toll int64) *models.Drive {
	a.t.Helper()
	ctx := context.Background()
	start := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	d := &models.Drive{VehicleID: a.vid, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: &distanceKm, DurationMin: 60}
	if err := a.repo.CreateManualDrive(ctx, d); err != nil {
		a.t.Fatal(err)
	}
	if toll > 0 {
		a.addToll(d, toll)
	}
	return d
}

func (a *carpoolAPI) addToll(d *models.Drive, cents int64) {
	a.t.Helper()
	exp := &models.DriveExpense{VehicleID: a.vid, DriveID: &d.ID, Type: "TOLL", Amount: money.Cents(cents), Currency: "EUR", Date: d.StartTime}
	if err := a.repo.SaveDriveExpense(context.Background(), exp, nil, ""); err != nil {
		a.t.Fatal(err)
	}
}

func legOf(trip map[string]any, i int) map[string]any {
	return trip["legs"].([]any)[i].(map[string]any)
}

func tollOf(m map[string]any) float64 { return m["tolls_cost"].(float64) }

func TestCarpoolLegsFollowTheirDrive(t *testing.T) {
	a := newCarpoolAPI(t, "follow")
	d := a.driveWithToll(100, 800)
	body := fmt.Sprintf(`{"title":"Trip","date":"2026-09-05","legs":[{"drive_id":%q,"distance_km":1,"tolls_cost":99,"electricity_cost":99}],`+
		`"passengers":[{"passenger_name":"Zoe","amount_paid":4}]}`, d.ID)

	trip := a.expect(http.StatusCreated, "POST", "/carpools", body)
	id := trip["id"].(string)
	leg := legOf(trip, 0)
	if tollOf(leg) != 8 || leg["electricity_cost"].(float64) == 99 || leg["distance_km"].(float64) != 100 {
		t.Fatalf("a leg made from a drive takes the drive's distance and cost, not the posted ones: %v", leg)
	}
	if dc, _ := leg["drive_cost"].(map[string]any); tollOf(dc) != 8 {
		t.Fatalf("drive_cost: %v", leg["drive_cost"])
	}
	if trip["costs_frozen_at"] != nil {
		t.Fatalf("a new carpool follows its drives: %v", trip["costs_frozen_at"])
	}

	// A toll added to the drive afterwards reaches the carpool page, its stored totals and the summary.
	a.addToll(d, 250)
	got := a.expect(http.StatusOK, "GET", "/carpools/"+id, "")
	if tollOf(legOf(got, 0)) != 10.5 || got["total_cost"].(float64) != 10.5 {
		t.Fatalf("the leg follows a toll added to its drive: %v", got)
	}
	list := a.expect(http.StatusOK, "GET", "/carpools", "")
	if sum := list["summary"].(map[string]any); sum["total_real_cost"].(float64) != 10.5 {
		t.Fatalf("the summary sums the refreshed amounts: %v", sum)
	}
	if stored, err := a.repo.GetCarpoolTrip(context.Background(), id, a.vid); err != nil || stored.TotalCost != 1050 {
		t.Fatalf("the refresh is stored: %v %v", stored, err)
	}
}

func TestCarpoolOverridesAndFreeze(t *testing.T) {
	a := newCarpoolAPI(t, "freeze")
	d := a.driveWithToll(100, 800)
	base := fmt.Sprintf(`"title":"Trip","date":"2026-09-05","passengers":[{"passenger_name":"Zoe"}],"legs":[{"drive_id":%q`, d.ID)

	a.expect(http.StatusBadRequest, "POST", "/carpools", `{`+base+`,"overrides":{"nope":1}}]}`)
	a.expect(http.StatusBadRequest, "POST", "/carpools", `{`+base+`,"overrides":{"tolls_cost":-1}}]}`)

	trip := a.expect(http.StatusCreated, "POST", "/carpools", `{`+base+`,"overrides":{"tolls_cost":5}}]}`)
	id := trip["id"].(string)
	if leg := legOf(trip, 0); tollOf(leg) != 5 || tollOf(leg["drive_cost"].(map[string]any)) != 8 || leg["overrides"].(map[string]any)["tolls_cost"].(float64) != 5 {
		t.Fatalf("an overridden category keeps its amount next to the drive's: %v", leg)
	}
	a.addToll(d, 200)
	if got := a.expect(http.StatusOK, "GET", "/carpools/"+id, ""); tollOf(legOf(got, 0)) != 5 || tollOf(legOf(got, 0)["drive_cost"].(map[string]any)) != 10 {
		t.Fatalf("an override survives a change of the drive: %v", legOf(got, 0))
	}

	frozen := a.expect(http.StatusOK, "POST", "/carpools/"+id+"/freeze", "")
	if frozen["costs_frozen_at"] == nil {
		t.Fatalf("freeze: %v", frozen)
	}
	a.addToll(d, 300)
	if got := a.expect(http.StatusOK, "GET", "/carpools/"+id, ""); tollOf(legOf(got, 0)) != 5 || tollOf(legOf(got, 0)["drive_cost"].(map[string]any)) != 13 {
		t.Fatalf("a frozen carpool keeps its amounts and still shows the drive's: %v", legOf(got, 0))
	}

	unfrozen := a.expect(http.StatusOK, "POST", "/carpools/"+id+"/unfreeze", "")
	if unfrozen["costs_frozen_at"] != nil || tollOf(legOf(unfrozen, 0)) != 5 {
		t.Fatalf("unfreeze follows the drive again, overrides included: %v", unfrozen)
	}

	// Saving with costs_frozen keeps the amounts as sent.
	saved := a.expect(http.StatusOK, "PUT", "/carpools/"+id, `{`+base+`,"distance_km":100,"tolls_cost":7}],"costs_frozen":true}`)
	if tollOf(legOf(saved, 0)) != 7 || saved["costs_frozen_at"] == nil {
		t.Fatalf("a frozen save stores the amounts as sent: %v", saved)
	}
	a.as(a.viewerID, http.StatusForbidden, "POST", "/carpools/"+id+"/freeze", "")
	a.expect(http.StatusNotFound, "POST", "/carpools/00000000-0000-0000-0000-000000000000/freeze", "")
}

func TestCarpoolKeepsItsAmountsWhenTheDriveIsDeleted(t *testing.T) {
	a := newCarpoolAPI(t, "deleted")
	d := a.driveWithToll(100, 800)
	trip := a.expect(http.StatusCreated, "POST", "/carpools", fmt.Sprintf(`{"title":"Trip","legs":[{"drive_id":%q}]}`, d.ID))
	id := trip["id"].(string)
	a.addToll(d, 200)
	a.expect(http.StatusOK, "GET", "/carpools/"+id, "")

	if err := a.repo.DeleteManualDrive(context.Background(), d.ID, a.vid); err != nil {
		t.Fatal(err)
	}
	got := a.expect(http.StatusOK, "GET", "/carpools/"+id, "")
	if leg := legOf(got, 0); leg["drive_id"] != nil || tollOf(leg) != 10 || got["total_cost"].(float64) != 10 {
		t.Fatalf("a leg whose drive is gone is manual and keeps the last amounts: %v", leg)
	}
}

func TestIntegrationCarpoolShowsTheCostOfTheDrives(t *testing.T) {
	a := newCarpoolAPI(t, "integration")
	d := a.driveWithToll(100, 800)
	a.expect(http.StatusCreated, "POST", "/carpools", fmt.Sprintf(`{"title":"Trip","legs":[{"drive_id":%q}]}`, d.ID))
	a.addToll(d, 200)

	h := NewIntegrationReadHandler(a.repo, nil).WithCarpool(services.NewCarpoolService(a.repo.Pool(), a.repo))
	out, err := h.Carpool(context.Background(), a.userID, a.vid, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary.RealCost.Amount != "10.00" || out.Trips[0].Tolls.Amount != "10.00" {
		t.Fatalf("integration readers see the cost of the drives, not the amounts stored at save: %+v", out)
	}
}
