package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

type whoAmIFixture struct {
	repo         *database.Repository
	owner, other *models.User
	daily, van   *models.Vehicle
	theirs       *models.Vehicle
	get          func(path, scope string, vehicles []string) *httptest.ResponseRecorder
}

func newWhoAmIFixture(t *testing.T) *whoAmIFixture {
	t.Helper()
	repo := authTestRepo(t)
	ctx := context.Background()
	f := &whoAmIFixture{repo: repo}
	var err error
	if f.owner, err = repo.CreateUser(ctx, "whoami-owner@example.com", "hash"); err != nil {
		t.Fatal(err)
	}
	if f.other, err = repo.CreateUser(ctx, "whoami-other@example.com", "hash"); err != nil {
		t.Fatal(err)
	}
	newVehicle := func(u *models.User, name, currency string) *models.Vehicle {
		v := &models.Vehicle{UserID: u.ID, Name: name, Currency: currency, CurrentOdometer: 10000, TeslaMateAuthType: models.AuthModeNone}
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	f.daily, f.van, f.theirs = newVehicle(f.owner, "Daily", "EUR"), newVehicle(f.owner, "Van", "GBP"), newVehicle(f.other, "Not mine", "USD")
	serviced, longAgo := 8000.0, 2000.0
	for _, rem := range []*models.MaintenanceReminder{
		{VehicleID: f.daily.ID, Title: "Oil", Category: "MAINTENANCE", IntervalKm: intPtr(5000), LastServiceOdometer: &longAgo},
		{VehicleID: f.van.ID, Title: "Brakes", Category: "MAINTENANCE", IntervalKm: intPtr(30000), LastServiceOdometer: &serviced},
		{VehicleID: f.theirs.ID, Title: "Not mine", Category: "OTHER", IntervalMonths: intPtr(12)},
	} {
		if err := repo.CreateMaintenanceReminder(ctx, rem); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.CreatePendingCharge(ctx, &models.PendingCharge{UserID: f.owner.ID, Source: "homeassistant", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now(), EnergyKwh: 12}); err != nil {
		t.Fatal(err)
	}

	h := NewIntegrationReadHandler(repo, services.NewTCOService(repo.Pool(), "UTC")).WithVersion("1.2.3")
	router := chi.NewRouter()
	router.Get("/whoami", h.GetWhoAmI)
	router.Get("/reminders", h.GetFleetReminders)
	router.Get("/summary", h.GetSummary)
	f.get = func(path, scope string, vehicles []string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		reqCtx := context.WithValue(req.Context(), middleware.UserIDKey, f.owner.ID)
		if scope != "" {
			reqCtx = context.WithValue(reqCtx, middleware.TokenScopeKey, scope)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(middleware.WithVehicleScope(reqCtx, vehicles)))
		return rec
	}
	return f
}

func intPtr(v int) *int { return &v }

func TestIntegrationWhoAmITellsTheScopeTheAccountAndThePendingCharges(t *testing.T) {
	f := newWhoAmIFixture(t)

	rec := f.get("/whoami", "read", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var out IntegrationWhoAmI
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Scope != "read" || out.Name != "whoami-owner@example.com" || out.VehiclesLimited || out.ServerVersion != "1.2.3" || out.PendingCharges != 1 {
		t.Errorf("credential: %+v", out)
	}
	if out.DistanceUnit == "" || out.ReportingZone != "UTC" || out.MonthInProgress == "" || len(out.Vehicles) != 2 {
		t.Errorf("account: %+v", out.IntegrationAccount)
	}
	currencies := map[string]string{}
	for _, v := range out.Vehicles {
		currencies[v.Name] = v.Currency
	}
	if currencies["Daily"] != "EUR" || currencies["Van"] != "GBP" {
		t.Errorf("currency per vehicle: %v", currencies)
	}
	if strings.Contains(rec.Body.String(), f.theirs.ID) {
		t.Errorf("another account's vehicle leaked: %s", rec.Body.String())
	}

	if err := json.Unmarshal(f.get("/whoami", "", nil).Body.Bytes(), &out); err != nil || out.Scope != "session" {
		t.Errorf("a session is not a token: %q %v", out.Scope, err)
	}

	limited := f.get("/whoami", "homeassistant", []string{f.van.ID})
	out = IntegrationWhoAmI{}
	if err := json.Unmarshal(limited.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Scope != "homeassistant" || !out.VehiclesLimited || len(out.Vehicles) != 1 || out.Vehicles[0].ID != f.van.ID {
		t.Errorf("a limited grant only sees its vehicles: %+v", out)
	}
}

func TestIntegrationFleetRemindersCoverTheVehiclesTheTokenSees(t *testing.T) {
	f := newWhoAmIFixture(t)
	read := func(path string, vehicles []string) IntegrationFleetReminders {
		t.Helper()
		rec := f.get(path, "read", vehicles)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), f.theirs.ID) || strings.Contains(rec.Body.String(), "Not mine") {
			t.Fatalf("%s leaked another account's reminder: %s", path, rec.Body.String())
		}
		var out IntegrationFleetReminders
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	all := read("/reminders", nil)
	if len(all.Vehicles) != 2 || all.Total != 2 || all.Overdue != 1 {
		t.Errorf("both vehicles, the oil change is overdue (due at 7,000 km, the van is at 10,000) and the brakes are not: %+v", all)
	}
	for _, v := range all.Vehicles {
		if v.VehicleID != f.daily.ID && v.VehicleID != f.van.ID {
			t.Errorf("unexpected vehicle %s", v.VehicleID)
		}
	}

	limited := read("/reminders", []string{f.van.ID})
	if len(limited.Vehicles) != 1 || limited.Vehicles[0].VehicleID != f.van.ID || limited.Total != 1 {
		t.Errorf("a limited grant only reads its vehicles: %+v", limited)
	}

	if rec := f.get("/reminders?status=soon", "read", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("an unknown status: %d, want 400", rec.Code)
	}
}
