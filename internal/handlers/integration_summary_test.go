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
	"github.com/teslacost/teslacost/internal/money"
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

	ev := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "ev", Currency: "EUR", Powertrain: models.PowertrainEV}, sum, stats, nil)
	if ev.ConsumptionKwh100km == nil || *ev.ConsumptionKwh100km != 17.5 || ev.ConsumptionL100km != nil {
		t.Errorf("an electric vehicle reports kWh only, even when the fuel figure is around: %+v", ev)
	}
	ice := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "ice", Currency: "EUR", Powertrain: models.PowertrainICE}, sum, nil, nil)
	if ice.ConsumptionL100km == nil || *ice.ConsumptionL100km != 6.4 || ice.ConsumptionKwh100km != nil {
		t.Errorf("a fuel vehicle reports litres only: %+v", ice)
	}
	phev := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "phev", Currency: "EUR", Powertrain: models.PowertrainPHEV}, sum, stats, nil)
	if phev.ConsumptionKwh100km == nil || phev.ConsumptionL100km == nil {
		t.Errorf("a plug-in hybrid reports both: %+v", phev)
	}
	none := buildIntegrationSummaryVehicle(&models.Vehicle{ID: "new", Currency: "EUR", Powertrain: models.PowertrainEV}, &services.TCOSummary{CurrentMonth: "2026-11"}, &services.EnergyStats{}, nil)
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

func TestIntegrationSummaryCarriesTheLastChargeTheCostPer100kmAndTheMonthSplit(t *testing.T) {
	f := newWhoAmIFixture(t)
	ctx := context.Background()
	started := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Second)
	ended := started.Add(45 * time.Minute)
	cost, zero := money.Cents(740), money.Cents(0)
	for _, c := range []*models.ChargeLog{
		{VehicleID: f.daily.ID, Date: started.Add(-48 * time.Hour), KwhAdded: 10, Cost: &zero, Currency: "EUR"},
		{VehicleID: f.daily.ID, Date: started, EndDate: &ended, KwhAdded: 31.5, Cost: &cost, Currency: "EUR"},
	} {
		if err := f.repo.CreateManualCharge(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.repo.CreateMaintenanceExpense(ctx, &models.MaintenanceExpense{
		VehicleID: f.daily.ID, Category: "MAINTENANCE", Amount: 4250, Currency: "EUR",
		Date: time.Now().UTC(), Description: "Service", AmortizationMode: "NONE",
	}); err != nil {
		t.Fatal(err)
	}

	out, _ := readSummary(t, f, "homeassistant", nil)
	byName := map[string]IntegrationSummaryVehicle{}
	for _, v := range out.Vehicles {
		byName[v.Name] = v
	}
	daily, van := byName["Daily"], byName["Van"]

	if daily.LastCharge == nil {
		t.Fatalf("the latest charge is missing: %+v", daily)
	}
	if daily.LastCharge.Date != started.Format(time.RFC3339) || daily.LastCharge.EnergyKwh != 31.5 {
		t.Errorf("the latest charge is the most recent one: %+v", daily.LastCharge)
	}
	if daily.LastCharge.Cost == nil || daily.LastCharge.Cost.Amount != "7.40" || daily.LastCharge.Cost.Currency != "EUR" {
		t.Errorf("its cost keeps the vehicle's currency: %+v", daily.LastCharge.Cost)
	}
	if daily.LastCharge.DurationMinutes == nil || *daily.LastCharge.DurationMinutes != 45 {
		t.Errorf("its duration is the length of the session: %+v", daily.LastCharge.DurationMinutes)
	}
	if van.LastCharge != nil {
		t.Errorf("a vehicle without a charge has none, not an empty one: %+v", van.LastCharge)
	}

	if daily.MonthCosts.Maintenance.Amount != "42.50" || daily.MonthCosts.Total.Amount != daily.MonthCost.Amount {
		t.Errorf("the month split adds up to the month cost: %+v / %s", daily.MonthCosts, daily.MonthCost.Amount)
	}
	if van.MonthCosts.Total.Amount != "0.00" || van.MonthCosts.Total.Currency != "GBP" {
		t.Errorf("a month without any cost splits into zeros of the vehicle's currency: %+v", van.MonthCosts)
	}

	if daily.LastCharge.PricePerKwh == nil || daily.LastCharge.PricePerKwh.Amount != "0.2349" {
		t.Errorf("the price per kWh is the cost over the energy added: %+v", daily.LastCharge.PricePerKwh)
	}
	if daily.MonthNet.Amount != daily.MonthCost.Amount || daily.CarpoolRevenueMonth.Amount != "0.00" {
		t.Errorf("without carpooling the net month is the month cost: %s / %s / %s", daily.MonthNet.Amount, daily.MonthCost.Amount, daily.CarpoolRevenueMonth.Amount)
	}
	if daily.Last12MonthsCost.Amount == "" || daily.Last12MonthsCost.Currency != "EUR" {
		t.Errorf("the twelve-month total keeps the vehicle's currency: %+v", daily.Last12MonthsCost)
	}
	if daily.MonthDistance.Unit != "km" || van.MonthDistance.Value != 0 || van.Last12MonthsCost.Currency != "GBP" {
		t.Errorf("a vehicle without any record has a zero distance and a zero total of its currency: %+v / %+v", van.MonthDistance, van.Last12MonthsCost)
	}
}

func TestIntegrationSummaryLastChargeKeepsWhatIsUnknownUnknown(t *testing.T) {
	started := time.Date(2026, 11, 4, 8, 0, 0, 0, time.UTC)
	if got := summaryLastCharge(nil, "EUR"); got != nil {
		t.Errorf("no charge is null: %+v", got)
	}
	open := summaryLastCharge(&models.ChargeLog{Date: started, KwhAdded: 12}, "EUR")
	if open.Cost != nil || open.DurationMinutes != nil {
		t.Errorf("an unpriced session without an end has neither cost nor duration: %+v", open)
	}
	zero := money.Cents(0)
	free := summaryLastCharge(&models.ChargeLog{Date: started, KwhAdded: 12, Cost: &zero}, "EUR")
	if free.Cost == nil || free.Cost.Amount != "0.00" {
		t.Errorf("a known free session costs zero: %+v", free.Cost)
	}
	if free.PricePerKwh == nil || free.PricePerKwh.Amount != "0.0000" {
		t.Errorf("a known free session costs nothing per kWh: %+v", free.PricePerKwh)
	}
	if open.PricePerKwh != nil {
		t.Errorf("an unknown cost has no price per kWh: %+v", open.PricePerKwh)
	}
	cost := money.Cents(500)
	noEnergy := summaryLastCharge(&models.ChargeLog{Date: started, KwhAdded: 0, Cost: &cost}, "EUR")
	if noEnergy.PricePerKwh != nil {
		t.Errorf("a session without energy has no price per kWh, not a division by zero: %+v", noEnergy.PricePerKwh)
	}
	before := started.Add(-time.Hour)
	backwards := summaryLastCharge(&models.ChargeLog{Date: started, EndDate: &before, KwhAdded: 12}, "EUR")
	if backwards.DurationMinutes == nil || *backwards.DurationMinutes != 0 {
		t.Errorf("an end before the start is a zero duration, never negative: %+v", backwards.DurationMinutes)
	}
}

func TestIntegrationSummaryCostPer100kmFollowsTheEnergyOfTheVehicle(t *testing.T) {
	cur := "EUR"
	measured := &IntegrationAmount{Amount: "3.10", Currency: cur}
	fuel := &IntegrationAmount{Amount: "9.20", Currency: cur}
	kwh, price := 16.0, 0.25
	profile := &models.Vehicle{Currency: cur, Powertrain: models.PowertrainEV, EstimatedKwh100km: &kwh, EstimatedPricePerKwh: &price}

	got := summaryCostPer100km(profile, IntegrationEnergyEfficiency{Electric: &IntegrationElectricEfficiency{CostPer100km: measured}, Fuel: &IntegrationFuelEfficiency{CostPer100km: fuel}})
	if got != measured {
		t.Errorf("the measured electric figure comes first: %+v", got)
	}
	got = summaryCostPer100km(profile, IntegrationEnergyEfficiency{Electric: &IntegrationElectricEfficiency{}})
	if got == nil || got.Amount != "4.00" || got.Currency != cur {
		t.Errorf("the profile estimate stands in while no distance is tracked: %+v", got)
	}
	ice := &models.Vehicle{Currency: cur, Powertrain: models.PowertrainICE, EstimatedKwh100km: &kwh, EstimatedPricePerKwh: &price}
	got = summaryCostPer100km(ice, IntegrationEnergyEfficiency{Fuel: &IntegrationFuelEfficiency{CostPer100km: fuel}})
	if got != fuel {
		t.Errorf("a vehicle that cannot be plugged in uses its fuel figure, not an electric estimate: %+v", got)
	}
	if got = summaryCostPer100km(&models.Vehicle{Currency: cur, Powertrain: models.PowertrainEV}, IntegrationEnergyEfficiency{}); got != nil {
		t.Errorf("nothing known is null, not zero: %+v", got)
	}
}
