package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func TestChargeEstimateRouteAnswersTheOwnerAndRefusesWhatItCannotPrice(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "estimate-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := repo.CreateUser(ctx, "estimate-stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	rate := money.Rate(250000)
	plan := &models.TariffPlan{UserID: owner.ID, Name: "Flat", PlanType: "FLAT", Currency: "EUR", FlatRateCents: &rate, IsDefault: true}
	if err := repo.CreateTariffPlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	h := NewIntegrationReadHandler(repo, nil).WithTariffs(services.NewTariffServiceIn("UTC"))

	get := func(serve http.HandlerFunc, userID string, q url.Values) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
		rec := httptest.NewRecorder()
		serve(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}

	status, body := get(h.GetChargeEstimate, owner.ID, url.Values{"energy_kwh": {"20"}, "duration_minutes": {"90"}, "start": {"2026-07-10T20:00:00Z"}})
	cost, _ := body["cost"].(map[string]any)
	if status != http.StatusOK || body["priced"] != true || cost["amount"] != "5.00" || cost["currency"] != "EUR" {
		t.Fatalf("a flat plan prices 20 kWh at 0.25: %d %v", status, body)
	}
	status, body = get(h.GetChargeEstimate, stranger.ID, url.Values{"energy_kwh": {"20"}, "duration_minutes": {"90"}})
	if status != http.StatusOK || body["priced"] != false || body["cost"] != nil {
		t.Fatalf("an account without a plan gets no price, not another account's: %d %v", status, body)
	}
	for name, q := range map[string]url.Values{
		"missing energy":  {"duration_minutes": {"90"}},
		"text duration":   {"energy_kwh": {"20"}, "duration_minutes": {"long"}},
		"zone-less start": {"energy_kwh": {"20"}, "duration_minutes": {"90"}, "start": {"2026-07-10T20:00:00"}},
		"above the cap":   {"energy_kwh": {"5000"}, "duration_minutes": {"90"}},
		"another's plan":  {"energy_kwh": {"20"}, "duration_minutes": {"90"}, "plan_id": {plan.ID}},
	} {
		user := owner.ID
		if name == "another's plan" {
			user = stranger.ID
		}
		if status, _ := get(h.GetChargeEstimate, user, q); status != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, status)
		}
	}

	status, body = get(h.GetTariffPlans, stranger.ID, nil)
	if plans, _ := body["plans"].([]any); status != http.StatusOK || plans == nil || len(plans) != 0 {
		t.Errorf("another account lists none of the plans: %d %v", status, body)
	}
}
