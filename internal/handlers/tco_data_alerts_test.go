package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestDataAlertsAreForTheVehiclesOwnerOnly(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "alerts-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "alerts-other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{UserID: owner.ID, Name: "Daily", Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	h := NewTCOHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	router := chi.NewRouter()
	router.Get("/{vehicleId}/data-alerts", h.GetDataAlerts)

	get := func(userID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/"+v.ID+"/data-alerts", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID)))
		return rec
	}
	if rec := get(other.ID); rec.Code != http.StatusNotFound {
		t.Errorf("another account: status %d, want 404", rec.Code)
	}
	rec := get(owner.ID)
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"alerts\":[]}\n" {
		t.Errorf("owner: status %d, body %q (an empty list is [], not null)", rec.Code, rec.Body.String())
	}
}
