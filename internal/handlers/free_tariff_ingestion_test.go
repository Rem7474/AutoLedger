package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

type freeTariffAPI struct {
	repo    *database.Repository
	router  chi.Router
	userID  string
	vehicle *models.Vehicle
}

func newFreeTariffAPI(t *testing.T, currency, powertrain string) *freeTariffAPI {
	t.Helper()
	repo := authTestRepo(t)
	ctx := context.Background()
	user, err := repo.CreateUser(ctx, "free-tariff@example.org", "hash")
	if err != nil {
		t.Fatal(err)
	}
	vehicle := &models.Vehicle{UserID: user.ID, Name: "Hybrid", Powertrain: powertrain, Currency: currency}
	if err := repo.CreateVehicle(ctx, vehicle); err != nil {
		t.Fatal(err)
	}
	tariffs := services.NewTariffService()
	router := chi.NewRouter()
	th := NewTariffHandler(repo, tariffs)
	ha := NewHomeAssistantHandler(repo, tariffs)
	pending := NewPendingChargesHandler(repo, tariffs)
	router.Post("/plans", th.Create)
	router.Post("/calculate", th.Calculate)
	router.Post("/event", ha.HandleEvent)
	router.Post("/pending/{id}/assign", pending.Assign)
	return &freeTariffAPI{repo: repo, router: router, userID: user.ID, vehicle: vehicle}
}

func (a *freeTariffAPI) call(t *testing.T, method, path, body string, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, a.userID))
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestZeroTariffIsKnownForDirectAndPendingCharging(t *testing.T) {
	a := newFreeTariffAPI(t, "USD", models.PowertrainPHEV)
	a.call(t, "POST", "/plans", `{"name":"Free electricity","currency":"USD","plan_type":"FLAT","flat_rate_cents":0,"is_default":true}`, 201)
	calc := a.call(t, "POST", "/calculate", `{"vehicle_id":"`+a.vehicle.ID+`","start_time":"2026-10-05T10:00:00Z","end_time":"2026-10-05T12:00:00Z","kwh":20}`, 200)
	if calc["cost"] != float64(0) || calc["plan"] != "Free electricity" {
		t.Fatalf("calc: %v", calc)
	}
	direct := a.call(t, "POST", "/event", `{"event_id":"free-direct","data":{"start_time":"2026-10-05T10:00:00Z","end_time":"2026-10-05T12:00:00Z","energy_kwh":20}}`, 201)
	if direct["cost"] != float64(0) {
		t.Fatalf("expected a known free cost, got %v", direct)
	}
	second := &models.Vehicle{UserID: a.userID, Name: "Second car", Powertrain: models.PowertrainEV, Currency: "USD"}
	if err := a.repo.CreateVehicle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	pending := a.call(t, "POST", "/event", `{"event_id":"free-pending","data":{"start_time":"2026-10-06T10:00:00Z","end_time":"2026-10-06T12:00:00Z","energy_kwh":20}}`, 201)
	assigned := a.call(t, "POST", "/pending/"+pending["pending_id"].(string)+"/assign", `{"vehicle_id":"`+a.vehicle.ID+`"}`, 200)
	if assigned["cost"] != float64(0) {
		t.Fatalf("expected a known free cost, got %v", assigned)
	}
	explicit := a.call(t, "POST", "/event", `{"vehicle_id":"`+a.vehicle.ID+`","event_id":"free-explicit","data":{"start_time":"2026-10-07T10:00:00Z","end_time":"2026-10-07T12:00:00Z","energy_kwh":20,"cost":0}}`, 201)
	if explicit["cost"] != float64(0) {
		t.Fatalf("explicit zero: %v", explicit)
	}
	count, err := a.repo.CountChargesWithoutCost(context.Background(), a.vehicle.ID)
	if err != nil || count != 0 {
		t.Fatalf("missing costs=%d, %v", count, err)
	}

}
