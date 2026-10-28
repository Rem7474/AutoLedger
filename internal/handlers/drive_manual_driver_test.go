package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// A manual drive can only be given a driver of its own vehicle: a person of another vehicle is refused, as on create
// and on the dedicated driver endpoints.
func TestUpdateManualDriveRefusesADriverOfAnotherVehicle(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "manual-drive-driver@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	kwh100 := 20.0
	v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "EV", TeslaMateAuthType: models.AuthModeNone, EstimatedKwh100km: &kwh100}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	other := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "Other", TeslaMateAuthType: models.AuthModeNone, EstimatedKwh100km: &kwh100}
	if err := repo.CreateVehicle(ctx, other); err != nil {
		t.Fatal(err)
	}
	// The first person of a vehicle becomes its default driver, which new drives inherit.
	if _, err := repo.CreateVehiclePerson(ctx, v.ID, "Default"); err != nil {
		t.Fatal(err)
	}
	own, err := repo.CreateVehiclePerson(ctx, v.ID, "Alice")
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := repo.CreateVehiclePerson(ctx, other.ID, "Bob")
	if err != nil {
		t.Fatal(err)
	}

	h := NewDriveHandler(repo, nil, nil)
	router := chi.NewRouter()
	router.Post("/vehicles/{vehicleId}/drives", h.Create)
	router.Put("/vehicles/{vehicleId}/drives/{driveId}", h.Update)
	send := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, u.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := send(http.MethodPost, "/vehicles/"+v.ID+"/drives", `{"start_time":"2026-09-01T08:00:00Z","distance_km":10}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body.String())
	}
	var d models.Drive
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	path := "/vehicles/" + v.ID + "/drives/" + d.ID
	driverOf := func() *string {
		t.Helper()
		var got *string
		if err := repo.Pool().QueryRow(ctx, `SELECT driver_id::text FROM drives WHERE id::text = $1`, d.ID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := driverOf()

	if rec := send(http.MethodPut, path, `{"distance_km":12,"driver_id":"`+stranger.ID+`"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("a driver of another vehicle must be refused, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := driverOf(); (got == nil) != (before == nil) || (got != nil && *got != *before) {
		t.Fatalf("the refused update changed the driver: %v -> %v", before, got)
	}

	if rec := send(http.MethodPut, path, `{"distance_km":12,"driver_id":"`+own.ID+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("a driver of the vehicle must be accepted, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := driverOf(); got == nil || *got != own.ID {
		t.Fatalf("driver = %v, want %s", got, own.ID)
	}
}
