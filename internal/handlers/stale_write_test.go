package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

// staleCase describes one versioned item: how to create it and how to save it again. body holds %V% where the
// expected_version member goes (nothing when the client sends none).
type staleCase struct {
	name       string
	createPath string
	createBody string
	putPath    string // %ID% is the created item's id
	body       string
	ghostPath  string
}

func withExpected(body string, version *int) string {
	extra := ""
	if version != nil {
		extra = fmt.Sprintf(`,"expected_version":%d`, *version)
	}
	return strings.ReplaceAll(body, "%V%", extra)
}

// TestVersionedItemsRefuseAnOutOfDateSave runs the same story on every item that carries a version: a save made from
// the version just read goes through and advances it, a save made from an older read is refused with request.stale and
// leaves the newer change in place, a client that sends no version is not checked, and a missing item is a 404.
func TestVersionedItemsRefuseAnOutOfDateSave(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "stale-items@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: "Car", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone, CurrentOdometer: 1000}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	var driveIDs []string
	for i := 0; i < 2; i++ {
		start := time.Date(2026, 3, 1+i, 8, 0, 0, 0, time.UTC)
		d := &models.Drive{VehicleID: v.ID, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: f64(10)}
		if err := repo.CreateManualDrive(ctx, d); err != nil {
			t.Fatal(err)
		}
		driveIDs = append(driveIDs, d.ID)
	}

	vehicles := NewVehicleHandler(repo, nil, nil)
	fuel, checkpoint := NewFuelHandler(repo), NewCheckpointHandler(repo)
	expenses, drives := NewExpenseHandler(repo, nil), NewDriveHandler(repo, nil, nil)
	r := chi.NewRouter()
	base := "/vehicles/{vehicleId}"
	r.Post(base+"/fuel-logs", fuel.Create)
	r.Put(base+"/fuel-logs/{fuelLogId}", fuel.Update)
	r.Post(base+"/odometer-checkpoints", checkpoint.Create)
	r.Put(base+"/odometer-checkpoints/{checkpointId}", checkpoint.Update)
	r.Post(base+"/maintenance", expenses.CreateMaintenance)
	r.Put(base+"/maintenance/{maintenanceId}", expenses.UpdateMaintenance)
	r.Post(base+"/expenses", expenses.CreateDriveExpense)
	r.Put(base+"/expenses/{expenseId}", expenses.UpdateDriveExpense)
	r.Get("/vehicles/{id}", vehicles.Get)
	r.Put("/vehicles/{id}", vehicles.Update)
	r.Post(base+"/drives", drives.Create)
	r.Put(base+"/drives/{driveId}", drives.Update)
	r.Post(base+"/charges", expenses.CreateManualCharge)
	r.Put(base+"/charges/{chargeId}", expenses.UpdateCharge)
	r.Post(base+"/trip-groups", drives.CreateTripGroup)
	r.Put(base+"/trip-groups/{groupId}", drives.UpdateTripGroup)
	a := &carpoolAPI{t: t, repo: repo, router: r, userID: owner.ID, vid: v.ID}

	const ghost = "00000000-0000-4000-8000-000000000000"
	cases := []staleCase{
		{"fuel log", "/fuel-logs", `{"date":"2026-03-01","amount":4000,"liters":40,"is_full_tank":true}`,
			"/fuel-logs/%ID%", `{"date":"2026-03-01","amount":4500,"liters":40,"is_full_tank":true%V%}`, "/fuel-logs/" + ghost},
		{"odometer reading", "/odometer-checkpoints", `{"date":"2026-02-01","odometer":900}`,
			"/odometer-checkpoints/%ID%", `{"date":"2026-02-01","odometer":901%V%}`, "/odometer-checkpoints/" + ghost},
		{"maintenance", "/maintenance", `{"category":"REPAIR","amount":10000,"date":"2026-03-01","description":"Oil"}`,
			"/maintenance/%ID%", `{"category":"REPAIR","amount":12000,"date":"2026-03-01","description":"Oil"%V%}`, "/maintenance/" + ghost},
		{"drive expense", "/expenses", `{"type":"TOLL","amount":500,"date":"2026-03-01","drive_id":"` + driveIDs[0] + `"}`,
			"/expenses/%ID%", `{"type":"TOLL","amount":600,"date":"2026-03-01","drive_id":"` + driveIDs[0] + `"%V%}`, "/expenses/" + ghost},
		{"trip group", "/trip-groups", `{"name":"Trip","drive_ids":["` + driveIDs[0] + `","` + driveIDs[1] + `"]}`,
			"/trip-groups/%ID%", `{"name":"Renamed"%V%}`, "/trip-groups/" + ghost},
		{"manual drive", "/drives", `{"start_time":"2026-03-05T08:00:00Z","end_time":"2026-03-05T09:00:00Z","distance_km":12}`,
			"/drives/%ID%", `{"start_time":"2026-03-05T08:00:00Z","end_time":"2026-03-05T09:00:00Z","distance_km":13%V%}`, "/drives/" + ghost},
		{"charge", "/charges", `{"date":"2026-03-06T10:00:00Z","kwh_added":10,"cost":500}`,
			"/charges/%ID%", `{"date":"2026-03-06T10:00:00Z","kwh_added":10,"cost":600%V%}`, "/charges/" + ghost},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			created := a.expect(http.StatusCreated, "POST", c.createPath, c.createBody)
			id := created["id"].(string)
			path := strings.ReplaceAll(c.putPath, "%ID%", id)
			loaded := int(created["version"].(float64))

			first := a.expect(http.StatusOK, "PUT", path, withExpected(c.body, &loaded))
			if int(first["version"].(float64)) != loaded+1 {
				t.Fatalf("a save advances the version by one: %v", first)
			}
			refused := a.expect(http.StatusConflict, "PUT", path, withExpected(c.body, &loaded))
			if refused["code"] != "request.stale" {
				t.Fatalf("a stale save answers request.stale: %v", refused)
			}
			current := loaded + 1
			a.expect(http.StatusOK, "PUT", path, withExpected(c.body, &current))
			a.expect(http.StatusOK, "PUT", path, withExpected(c.body, nil))
			a.expect(http.StatusNotFound, "PUT", c.ghostPath, withExpected(c.body, &loaded))
		})
	}
}

// TestVehicleRefusesAnOutOfDateSave covers the vehicle itself, which is read rather than created and has no vehicle
// prefix in its own path.
func TestVehicleRefusesAnOutOfDateSave(t *testing.T) {
	a := newCarpoolAPI(t, "staleVehicle")
	a.router.Get("/vehicles/{id}", NewVehicleHandler(a.repo, nil, nil).Get)
	a.router.Put("/vehicles/{id}", NewVehicleHandler(a.repo, nil, nil).Update)

	loaded := int(a.expect(http.StatusOK, "GET", "", "")["version"].(float64))
	first := a.expect(http.StatusOK, "PUT", "", withExpected(`{"name":"First editor"%V%}`, &loaded))
	if int(first["version"].(float64)) != loaded+1 {
		t.Fatalf("a save advances the version by one: %v", first)
	}
	refused := a.expect(http.StatusConflict, "PUT", "", withExpected(`{"name":"Second editor"%V%}`, &loaded))
	if refused["code"] != "request.stale" {
		t.Fatalf("a stale save answers request.stale: %v", refused)
	}
	if got := a.expect(http.StatusOK, "GET", "", ""); got["name"] != "First editor" {
		t.Fatalf("the first editor's change was overwritten: %v", got)
	}
	current := loaded + 1
	a.expect(http.StatusOK, "PUT", "", withExpected(`{"name":"Second editor"%V%}`, &current))
	a.expect(http.StatusOK, "PUT", "", withExpected(`{"name":"No version"%V%}`, nil))
}

// TestDriveVersionAdvancesOnUserWritesNotOnJobs: a tag, driver or toll-review change made by a user moves a drive's
// version (an editor opened before it must be refused); the geocoder filling an address does not.
func TestDriveVersionAdvancesOnUserWritesNotOnJobs(t *testing.T) {
	a := newCarpoolAPI(t, "driveVersion")
	ctx := context.Background()
	start := time.Date(2026, 3, 7, 8, 0, 0, 0, time.UTC)
	d := &models.Drive{VehicleID: a.vid, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: f64(10), StartAddress: strPtr("12.5,3.4")}
	if err := a.repo.CreateManualDrive(ctx, d); err != nil {
		t.Fatal(err)
	}
	version := func() int64 {
		got, err := a.repo.GetDriveByID(ctx, d.ID, a.vid)
		if err != nil {
			t.Fatal(err)
		}
		return got.Version
	}
	if version() != 1 {
		t.Fatalf("a new drive starts at version 1, got %d", version())
	}
	if err := a.repo.UpdateDriveTags(ctx, d.ID, a.vid, []string{"work"}); err != nil {
		t.Fatal(err)
	}
	if err := a.repo.SetDriveTollReviewed(ctx, d.ID, a.vid, true); err != nil {
		t.Fatal(err)
	}
	if err := a.repo.SetDriveDriver(ctx, d.ID, a.vid, nil); err != nil {
		t.Fatal(err)
	}
	if got := version(); got != 4 {
		t.Fatalf("each user write advances the version: want 4, got %d", got)
	}
	if err := a.repo.ReplaceDriveAddress(ctx, d.ID, database.DriveEndDeparture, "12.5,3.4", "1 Rue de Paris"); err != nil {
		t.Fatal(err)
	}
	if got := version(); got != 4 {
		t.Fatalf("geocoding must not advance the version, got %d", got)
	}
}
