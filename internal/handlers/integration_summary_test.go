package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
	"net/http/httptest"
)

func readSummary(t *testing.T, f *whoAmIFixture, scope string, vehicles []string) (IntegrationSummary, string) {
	t.Helper()
	rec := f.get("/summary", scope, vehicles)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var out IntegrationSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out, rec.Body.String()
}

func TestIntegrationSummaryGivesOneEntryPerVisibleVehicleInItsOwnCurrency(t *testing.T) {
	f := newWhoAmIFixture(t)
	expense := &models.MaintenanceExpense{
		VehicleID: f.daily.ID, Category: "MAINTENANCE", Amount: 4250, Currency: "EUR",
		Date: time.Now().UTC(), Description: "Service", AmortizationMode: "NONE",
	}
	if err := f.repo.CreateMaintenanceExpense(context.Background(), expense); err != nil {
		t.Fatal(err)
	}

	out, body := readSummary(t, f, "homeassistant", nil)
	if len(out.Vehicles) != 2 {
		t.Fatalf("want the 2 vehicles of the account, got %+v", out.Vehicles)
	}
	if strings.Contains(body, f.theirs.ID) || strings.Contains(body, "Not mine") {
		t.Errorf("another account's vehicle leaked: %s", body)
	}
	byName := map[string]IntegrationSummaryVehicle{}
	for _, v := range out.Vehicles {
		byName[v.Name] = v
	}
	daily, van := byName["Daily"], byName["Van"]
	if daily.Currency != "EUR" || daily.MonthCost.Currency != "EUR" || van.Currency != "GBP" || van.MonthCost.Currency != "GBP" {
		t.Errorf("each vehicle keeps its currency: %+v / %+v", daily, van)
	}
	if daily.Month == "" || daily.Month != van.Month {
		t.Errorf("month in progress: %q / %q", daily.Month, van.Month)
	}
	if daily.MonthCost.Amount != "42.50" {
		t.Errorf("month cost of the vehicle with an expense: %s", daily.MonthCost.Amount)
	}
	if daily.ConsumptionL100km != nil || daily.ConsumptionKwh100km != nil {
		t.Errorf("no drive or charge recorded, so no consumption: %+v", daily)
	}
	if van.MonthCost.Amount != "0.00" {
		t.Errorf("month cost of a vehicle without expense is zero, not unknown: %s", van.MonthCost.Amount)
	}
}

func TestIntegrationSummaryOfALimitedGrantOnlyCoversItsVehicles(t *testing.T) {
	f := newWhoAmIFixture(t)

	out, body := readSummary(t, f, "homeassistant", []string{f.van.ID})
	if len(out.Vehicles) != 1 || out.Vehicles[0].VehicleID != f.van.ID {
		t.Fatalf("a limited grant sees its vehicle only: %+v", out.Vehicles)
	}
	if strings.Contains(body, f.daily.ID) || strings.Contains(body, f.theirs.ID) {
		t.Errorf("a vehicle outside the grant leaked: %s", body)
	}
}

func TestIntegrationSummaryOfAnAccountWithoutVehicleIsAnEmptyList(t *testing.T) {
	f := newWhoAmIFixture(t)

	out, body := readSummary(t, f, "read", []string{"00000000-0000-0000-0000-000000000000"})
	if len(out.Vehicles) != 0 || !strings.Contains(body, `"vehicles":[]`) {
		t.Errorf("want an empty list, not null: %s", body)
	}
}

func TestIntegrationSummaryVehicleReportsTheConsumptionOfTheEnergyItUses(t *testing.T) {
	sum := &services.TCOSummary{CurrentMonth: "2026-11", ConsumptionL100km: f64(6.4)}
	stats := &services.EnergyStats{Summary: services.EnergySummary{ConsumptionKwh100km: f64(17.5)}}

	ev := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "ev", Currency: "EUR", Powertrain: models.PowertrainEV}, sum, stats)
	if ev.ConsumptionKwh100km == nil || *ev.ConsumptionKwh100km != 17.5 || ev.ConsumptionL100km != nil {
		t.Errorf("an electric vehicle reports kWh only, even when the fuel figure is around: %+v", ev)
	}
	ice := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "ice", Currency: "EUR", Powertrain: models.PowertrainICE}, sum, nil)
	if ice.ConsumptionL100km == nil || *ice.ConsumptionL100km != 6.4 || ice.ConsumptionKwh100km != nil {
		t.Errorf("a fuel vehicle reports litres only: %+v", ice)
	}
	phev := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "phev", Currency: "EUR", Powertrain: models.PowertrainPHEV}, sum, stats)
	if phev.ConsumptionKwh100km == nil || phev.ConsumptionL100km == nil {
		t.Errorf("a plug-in hybrid reports both: %+v", phev)
	}
	none := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "new", Currency: "EUR", Powertrain: models.PowertrainEV}, &services.TCOSummary{CurrentMonth: "2026-11"}, &services.EnergyStats{})
	if none.ConsumptionKwh100km != nil || none.ConsumptionL100km != nil || none.CostPerKm != nil {
		t.Errorf("no data is null, not zero: %+v", none)
	}
}

func TestIntegrationSummaryFailsWhenTheVehiclesCannotBeRead(t *testing.T) {
	f := newWhoAmIFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/summary", nil)
	ctx, cancel := context.WithCancel(context.WithValue(req.Context(), middleware.UserIDKey, f.owner.ID))
	cancel()
	h := NewIntegrationReadHandler(f.repo, services.NewTCOService(f.repo.Pool(), "UTC"))

	rec := httptest.NewRecorder()
	h.GetSummary(rec, req.WithContext(ctx))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("a failed read is an error, not a partial list: %d %s", rec.Code, rec.Body.String())
	}
}
