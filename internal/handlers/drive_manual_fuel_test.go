package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// A combustion vehicle enters a drive by hand: the drive carries no kWh, and its fuel is read back in litres from the
// consumption and the price of its fill-ups, left unknown while they give none.
func TestManualDriveOnCombustionVehicleCarriesFuelNotKwh(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "ice-manual-drive@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	kwh100 := 18.0 // never applied to a fuel-only vehicle
	v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "ICE", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone, EstimatedKwh100km: &kwh100}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}

	h := NewDriveHandler(repo, services.NewCarpoolService(repo.Pool(), repo), nil)
	router := chi.NewRouter()
	router.Post("/vehicles/{vehicleId}/drives", h.Create)
	router.Put("/vehicles/{vehicleId}/drives/{driveId}", h.Update)
	router.Get("/vehicles/{vehicleId}/drives", h.List)
	do := func(method, path, body string, want int, out any) {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, u.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s: status %d, body %s", method, path, rec.Code, rec.Body.String())
		}
		if out != nil {
			if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
				t.Fatal(err)
			}
		}
	}
	type listed struct {
		Drives []struct {
			ID    string             `json:"id"`
			Costs DriveCostBreakdown `json:"costs"`
		} `json:"drives"`
	}

	// A kWh typed by a client is ignored; nothing is estimated
	var created models.Drive
	do(http.MethodPost, "/vehicles/"+v.ID+"/drives", `{"start_time":"2026-09-01T08:00:00Z","distance_km":100,"energy_consumed_kwh":12}`, http.StatusCreated, &created)
	if created.EnergyConsumedKwh != nil || created.ConsumptionKwh100km != nil || created.EnergyEstimated || !created.IsManual {
		t.Fatalf("a combustion drive carries no kWh: %+v", created)
	}
	var edited models.Drive
	do(http.MethodPut, "/vehicles/"+v.ID+"/drives/"+created.ID, `{"distance_km":80,"energy_consumed_kwh":9}`, http.StatusOK, &edited)
	if edited.EnergyConsumedKwh != nil || edited.ConsumptionKwh100km != nil {
		t.Fatalf("an edit keeps the drive without kWh: %+v", edited)
	}

	// No fill-up yet: the fuel and its cost are unknown, not invented
	var l listed
	do(http.MethodGet, "/vehicles/"+v.ID+"/drives", "", http.StatusOK, &l)
	c := l.Drives[0].Costs
	if !c.FuelOnly || c.FuelLiters != nil || c.EnergySource != services.EnergySourceUnknown || c.ElectricityCost != 0 || c.ElectricityKwh != 0 || !c.HasEstimates {
		t.Fatalf("without fill-ups the fuel is unknown: %+v", c)
	}

	// Two full tanks 500 km apart with 40 L: 8 L/100 km at 1.50 EUR/L
	odo1, odo2, l1, l2 := 1000.0, 1500.0, 20.0, 40.0
	for _, f := range []models.FuelLog{
		{VehicleID: v.ID, Date: time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC), Odometer: &odo1, Amount: money.FromFloat(30), Liters: &l1, IsFullTank: true},
		{VehicleID: v.ID, Date: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC), Odometer: &odo2, Amount: money.FromFloat(60), Liters: &l2, IsFullTank: true},
	} {
		if err := repo.CreateFuelLog(ctx, &f); err != nil {
			t.Fatal(err)
		}
	}
	do(http.MethodGet, "/vehicles/"+v.ID+"/drives", "", http.StatusOK, &l)
	c = l.Drives[0].Costs
	if c.FuelLiters == nil || *c.FuelLiters != 6.4 || c.FuelRatePerLiter != 1.5 || c.EnergySource != services.EnergySourceFuel || c.ElectricityKwh != 0 {
		t.Fatalf("80 km at 8 L/100 km and 1.5 EUR/L: %+v", c)
	}
	if c.ElectricityCost != money.FromFloat(9.6) {
		t.Fatalf("fuel cost: got %v, want 9.60", c.ElectricityCost)
	}
}

// A hybrid keeps the electric figures: kWh estimated from its consumption, no fuel part invented.
func TestManualDriveOnHybridStaysElectric(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "phev-manual-drive@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	kwh100 := 20.0
	v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "PHEV", Powertrain: models.PowertrainPHEV, TeslaMateAuthType: models.AuthModeNone, EstimatedKwh100km: &kwh100}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	odo1, odo2, l1, l2 := 1000.0, 1500.0, 20.0, 40.0
	for _, f := range []models.FuelLog{
		{VehicleID: v.ID, Date: time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC), Odometer: &odo1, Amount: money.FromFloat(30), Liters: &l1, IsFullTank: true},
		{VehicleID: v.ID, Date: time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC), Odometer: &odo2, Amount: money.FromFloat(60), Liters: &l2, IsFullTank: true},
	} {
		if err := repo.CreateFuelLog(ctx, &f); err != nil {
			t.Fatal(err)
		}
	}
	h := NewDriveHandler(repo, services.NewCarpoolService(repo.Pool(), repo), nil)
	router := chi.NewRouter()
	router.Post("/vehicles/{vehicleId}/drives", h.Create)
	router.Get("/vehicles/{vehicleId}/drives", h.List)
	do := func(method, path, body string, out any) {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, u.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("%s %s: status %d, body %s", method, path, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	var created models.Drive
	do(http.MethodPost, "/vehicles/"+v.ID+"/drives", `{"start_time":"2026-09-01T08:00:00Z","distance_km":50}`, &created)
	if created.EnergyConsumedKwh == nil || *created.EnergyConsumedKwh != 10 || !created.EnergyEstimated {
		t.Fatalf("a hybrid drive estimates kWh from its consumption: %+v", created)
	}
	var l struct {
		Drives []struct {
			Costs DriveCostBreakdown `json:"costs"`
		} `json:"drives"`
	}
	do(http.MethodGet, "/vehicles/"+v.ID+"/drives", "", &l)
	if c := l.Drives[0].Costs; c.FuelOnly || c.FuelLiters != nil || c.ElectricityKwh != 10 {
		t.Fatalf("no fuel part on a hybrid drive: %+v", c)
	}
}
