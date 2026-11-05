package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func TestCurrentTariffGivesThePriceNowAndTheNextChange(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "current-tariff-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := repo.CreateUser(ctx, "current-tariff-stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	h := NewIntegrationReadHandler(repo, nil).WithTariffs(services.NewTariffServiceIn("UTC"))
	noon := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	none, err := h.CurrentTariff(ctx, owner.ID, "", noon)
	if err != nil {
		t.Fatal(err)
	}
	if none.Priced || none.Now != nil || none.NextChange != nil || none.Plan != nil || none.UnpricedReason == nil || *none.UnpricedReason == "" {
		t.Errorf("an account without a plan has no price, and says why: %+v", none)
	}

	night := &models.TariffPlan{UserID: owner.ID, Name: "Night", PlanType: "BANDS", Currency: "GBP", DefaultBand: "day", IsDefault: true,
		Bands: []models.TariffBand{{Name: "day", RateCents: 200000}, {Name: "night", RateCents: 105000}},
		Rules: []models.TariffRule{{Start: "22:00", End: "06:00", Band: "night"}}}
	if err := repo.CreateTariffPlan(ctx, night); err != nil {
		t.Fatal(err)
	}

	day, err := h.CurrentTariff(ctx, owner.ID, "", noon)
	if err != nil {
		t.Fatal(err)
	}
	if !day.Priced || day.UnpricedReason != nil || day.Plan == nil || day.Plan.ID != night.ID || day.Plan.Name != "Night" ||
		day.Now == nil || day.Now.Band != "day" || day.Now.PricePerKwh.Amount != "0.20" || day.Now.PricePerKwh.Currency != "GBP" {
		t.Fatalf("midday is the day band of the default plan: %+v", day)
	}
	if day.NextChange == nil || day.NextChange.At != "2026-07-10T22:00:00Z" || day.NextChange.Band != "night" || day.NextChange.PricePerKwh.Amount != "0.105" {
		t.Errorf("the night band starts at 22:00 with its six-decimal price kept: %+v", day.NextChange)
	}

	night2, err := h.CurrentTariff(ctx, owner.ID, "", time.Date(2026, 7, 10, 23, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if night2.Now.Band != "night" || night2.NextChange == nil || night2.NextChange.At != "2026-07-11T06:00:00Z" || night2.NextChange.Band != "day" {
		t.Errorf("at night the next change is the morning: %+v", night2)
	}

	if _, err := h.CurrentTariff(ctx, stranger.ID, "00000000-0000-0000-0000-000000000000", noon); err == nil {
		t.Error("another account's vehicle is refused")
	}
	other, err := h.CurrentTariff(ctx, stranger.ID, "", noon)
	if err != nil || other.Priced {
		t.Errorf("another account sees none of the owner's plans: %+v %v", other, err)
	}
}

func TestCurrentTariffOfAFlatPlanNeverChanges(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "current-tariff-flat@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	rate := money.Rate(250000)
	if err := repo.CreateTariffPlan(ctx, &models.TariffPlan{UserID: owner.ID, Name: "Flat", PlanType: "FLAT", Currency: "EUR", FlatRateCents: &rate, IsDefault: true}); err != nil {
		t.Fatal(err)
	}
	h := NewIntegrationReadHandler(repo, nil).WithTariffs(services.NewTariffServiceIn("UTC"))

	req := httptest.NewRequest(http.MethodGet, "/?"+url.Values{}.Encode(), nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, owner.ID))
	rec := httptest.NewRecorder()
	h.GetCurrentTariff(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	now, _ := body["now"].(map[string]any)
	price, _ := now["price_per_kwh"].(map[string]any)
	if rec.Code != http.StatusOK || body["priced"] != true || now["band"] != "FLAT" || price["amount"] != "0.25" || price["currency"] != "EUR" {
		t.Fatalf("a flat plan gives its price: %d %v", rec.Code, body)
	}
	if value, present := body["next_change"]; !present || value != nil {
		t.Errorf("a flat plan has no next change, as an explicit null: %v", body)
	}
}

func TestCurrentTariffNeedsTheTariffService(t *testing.T) {
	h := NewIntegrationReadHandler(authTestRepo(t), nil)
	if _, err := h.CurrentTariff(context.Background(), "someone", "", time.Now()); err == nil {
		t.Error("without the tariff service the price cannot be read")
	}
}
