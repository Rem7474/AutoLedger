package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestCostHistoryIsScopedToTheVehicleOwnerAndValidatesTheKind(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "history-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := repo.CreateUser(ctx, "history-stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: "EV", Powertrain: models.PowertrainEV, TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Pool().Exec(ctx, `INSERT INTO drive_expenses (vehicle_id, type, amount, date) VALUES ($1, 'TOLL', 12.5, NOW())`, v.ID); err != nil {
		t.Fatal(err)
	}

	h := NewTCOHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	router := chi.NewRouter()
	router.Get("/vehicles/{vehicleId}/cost-history", h.GetCostHistory)
	call := func(userID, query string) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/vehicles/"+v.ID+"/cost-history"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}

	status, body := call(owner.ID, "?kind=tolls")
	if status != http.StatusOK {
		t.Fatalf("owner: %d %v", status, body)
	}
	months := body["months"].([]any)
	if body["kind"] != "tolls" || len(months) == 0 || months[len(months)-1].(map[string]any)["amount"].(float64) != 12.5 {
		t.Fatalf("body = %v", body)
	}
	if status, _ := call(stranger.ID, "?kind=tolls"); status != http.StatusNotFound {
		t.Errorf("another user's vehicle must answer 404, got %d", status)
	}
	if status, _ := call(owner.ID, "?kind=energy"); status != http.StatusBadRequest {
		t.Errorf("an unknown kind must answer 400, got %d", status)
	}
	if status, _ := call(owner.ID, ""); status != http.StatusBadRequest {
		t.Errorf("a missing kind must answer 400, got %d", status)
	}
}
