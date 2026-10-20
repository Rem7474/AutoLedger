package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func monthsUpTo(last time.Time, n int) []services.MonthlyCost {
	out := make([]services.MonthlyCost, 0, n)
	for i := n - 1; i >= 0; i-- {
		m := time.Date(last.Year(), last.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		out = append(out, services.MonthlyCost{
			Month: m.Format("2006-01"), DistanceKm: 100,
			Energy: 1000, Tolls: 200, Maintenance: 300, Insurance: 400, Financing: 500, Tires: 600, Other: 700, Total: 3700,
		})
	}
	return out
}

func TestIntegrationCostsCoverTheLastTwelveMonthsAndSincePurchase(t *testing.T) {
	vehicle := &models.Vehicle{ID: "v1", Currency: "GBP"}
	perKm := 0.25
	sum := &services.TCOSummary{
		MonthlyCosts: monthsUpTo(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 30),
		TotalCost:    money.Cents(250000), TotalCostPerKm: &perKm, DistanceBasisKm: 10000,
		EnergyCost: 100000, TollsCost: 10000, MaintenanceCost: 40000, RepairCost: 5000, InsuranceCost: 60000,
		FinancingCost: 20000, TiresCost: 8000, SubscriptionCost: 3000, TaxCost: 2000, OtherCost: 2000,
		Completeness: services.TCOCompleteness{ScoreAvailable: true, ScorePct: 82, Warnings: nil},
	}

	got := buildIntegrationCosts(vehicle, sum)

	l12 := got.Last12Months
	if l12.From == nil || *l12.From != "2025-11" || *l12.To != "2026-10" {
		t.Errorf("the period must be the twelve months ending with the latest, got %v..%v", l12.From, l12.To)
	}
	if l12.Costs.Total != (IntegrationAmount{Amount: "444.00", Currency: "GBP"}) {
		t.Errorf("twelve months of 37.00 make 444.00 GBP, got %+v", l12.Costs.Total)
	}
	if l12.Distance != (IntegrationDistance{Value: 1200, Unit: "km"}) {
		t.Errorf("distance: %+v", l12.Distance)
	}
	if l12.CostPerKm == nil || l12.CostPerKm.Amount != "0.3700" {
		t.Errorf("cost per km is the total over the distance, got %+v", l12.CostPerKm)
	}
	cats := l12.Costs
	sumOfCategories := cats.Energy.Amount + "|" + cats.Tolls.Amount + "|" + cats.Maintenance.Amount + "|" + cats.Insurance.Amount + "|" +
		cats.Financing.Amount + "|" + cats.Tires.Amount + "|" + cats.Other.Amount
	if sumOfCategories != "120.00|24.00|36.00|48.00|60.00|72.00|84.00" {
		t.Errorf("categories: %s", sumOfCategories)
	}

	since := got.SincePurchase
	if since.From != nil || since.To != nil {
		t.Error("since purchase has no month bounds")
	}
	if since.Costs.Maintenance.Amount != "450.00" || since.Costs.Other.Amount != "70.00" || since.Costs.Total.Amount != "2500.00" {
		t.Errorf("repairs count with maintenance, subscriptions and taxes with other: %+v", since.Costs)
	}
	if since.CostPerKm == nil || since.CostPerKm.Amount != "0.2500" || since.CostPerKm.Currency != "GBP" {
		t.Errorf("since purchase per km: %+v", since.CostPerKm)
	}
	if got.Completeness.ScorePct == nil || *got.Completeness.ScorePct != 82 || got.Completeness.Warnings == nil {
		t.Errorf("completeness: %+v", got.Completeness)
	}
}

func TestIntegrationCostsSayUnknownWhenNothingIsKnown(t *testing.T) {
	got := buildIntegrationCosts(&models.Vehicle{ID: "v1", Currency: "EUR"}, &services.TCOSummary{})

	if got.Last12Months.From != nil || got.Last12Months.CostPerKm != nil || got.SincePurchase.CostPerKm != nil {
		t.Errorf("no month and no distance: the period and the per-km cost are null, got %+v", got)
	}
	if got.Completeness.ScorePct != nil {
		t.Error("an unavailable score is null, not 0")
	}
	body, _ := json.Marshal(got)
	if !strings.Contains(string(body), `"warnings":[]`) || !strings.Contains(string(body), `"cost_per_km":null`) {
		t.Errorf("empty lists are [] and unknown figures null: %s", body)
	}
}

func TestIntegrationDataGapsAreCappedAndCarryNoDriveIdentity(t *testing.T) {
	prev := "drive-0"
	issues := make([]models.DataQualityIssue, 0, integrationReadIssues+1)
	for i := 0; i <= integrationReadIssues; i++ {
		issues = append(issues, models.DataQualityIssue{Type: "ODOMETER_GAP", DriveID: fmt.Sprintf("drive-%d", i+1), PreviousDriveID: &prev,
			Date: time.Date(2026, 3, 1+i%27, 23, 30, 0, 0, time.FixedZone("x", 2*3600)), Km: 12.5})
	}
	comp := services.TCOCompleteness{ChargesWithoutCost: 3, UnqualifiedDrives: 2, UntrackedDistanceKm: 410, InsuranceMissing: true}

	got := buildIntegrationDataGaps("v1", "EUR", issues, 31, 640, 4, comp)

	if len(got.Issues) != integrationReadIssues || !got.Truncated {
		t.Errorf("%d issues, truncated %v: want %d and true", len(got.Issues), got.Truncated, integrationReadIssues)
	}
	if got.OdometerGaps != 31 || got.OdometerGapDistance.Value != 640 || got.OdometerAnomalies != 4 {
		t.Errorf("the counts are the full ones, not the capped list's: %+v", got)
	}
	if got.Issues[0].At != "2026-03-01T21:30:00Z" {
		t.Errorf("an instant is given in UTC, got %s", got.Issues[0].At)
	}
	body, _ := json.Marshal(got)
	if strings.Contains(string(body), "drive-") {
		t.Errorf("a drive identifier leaked: %s", body)
	}
	if got.ChargesWithoutCost != 3 || got.UntrackedDistance.Value != 410 || !got.InsuranceMissing || got.AcquisitionMissing {
		t.Errorf("completeness gaps: %+v", got)
	}

	if few := buildIntegrationDataGaps("v1", "EUR", issues[:2], 2, 1, 0, comp); few.Truncated || len(few.Issues) != 2 {
		t.Errorf("a short list is not truncated: %+v", few)
	}
}

func TestIntegrationDataGapsRankWhatToCompleteWithItsWorthAndUnit(t *testing.T) {
	estimate := money.Cents(4200)
	comp := services.TCOCompleteness{
		StartOdometerMissing: true, UnconvertedExpenses: 2, AcquisitionMissing: true,
		AcquisitionMissingFields: []string{"resale_value"},
		Gaps: []services.CompletenessGap{
			{Key: "depreciation"},
			{Key: "unpriced_charges", Count: 3, Quantity: 120.5, Amount: &estimate},
			{Key: "untracked_distance", Quantity: 410},
			{Key: "unconverted_expenses", Count: 2, Target: "expenses"},
		},
		Dimensions: []services.CompletenessDimension{{Key: "energy", Label: "Energy", ScorePct: 80, Weight: 0.4, Applicable: true}},
	}

	got := buildIntegrationDataGaps("v1", "CHF", nil, 0, 0, 0, comp)

	kinds := make([]string, len(got.Missing))
	for i, m := range got.Missing {
		kinds[i] = m.Kind
	}
	if strings.Join(kinds, ",") != "depreciation,unpriced_charges,untracked_distance,unconverted_expenses" {
		t.Errorf("the order of the application's list is kept: %v", kinds)
	}
	depreciation, charges, distance, unconverted := got.Missing[0], got.Missing[1], got.Missing[2], got.Missing[3]
	if depreciation.EstimatedAmount != nil || depreciation.Quantity != nil {
		t.Errorf("a depreciation is never estimated: %+v", depreciation)
	}
	if charges.EstimatedAmount == nil || *charges.EstimatedAmount != (IntegrationAmount{Amount: "42.00", Currency: "CHF"}) ||
		charges.Count != 3 || charges.Quantity == nil || *charges.Quantity != 120.5 || *charges.QuantityUnit != "kWh" {
		t.Errorf("unpriced charges carry their worth in the vehicle's currency and their energy: %+v", charges)
	}
	if distance.EstimatedAmount != nil || *distance.Quantity != 410 || *distance.QuantityUnit != "km" {
		t.Errorf("an unestimated gap has a null amount, not zero: %+v", distance)
	}
	if unconverted.FixIn == nil || *unconverted.FixIn != "expenses" || unconverted.Quantity != nil {
		t.Errorf("unconverted expenses say where they are fixed: %+v", unconverted)
	}
	if !got.StartOdometerMissing || got.UnconvertedExpenses != 2 || len(got.AcquisitionMissingFields) != 1 || len(got.ScoreDimensions) != 1 || !got.ScoreDimensions[0].Applicable {
		t.Errorf("the other completeness details: %+v", got)
	}

	empty := buildIntegrationDataGaps("v1", "CHF", nil, 0, 0, 0, services.TCOCompleteness{})
	body, _ := json.Marshal(empty)
	for _, want := range []string{`"missing":[]`, `"score_dimensions":[]`, `"acquisition_missing_fields":[]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("an empty list is [], not null (%s): %s", want, body)
		}
	}
}

func TestIntegrationReadRoutesServeOnlyTheAccountsVehiclesWithoutIdentifiers(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, _ := repo.CreateUser(ctx, "read-owner@example.com", "hash")
	other, _ := repo.CreateUser(ctx, "read-other@example.com", "hash")
	vin := "VF1SECRETVIN000001"
	mine := &models.Vehicle{UserID: owner.ID, Name: "Daily", Make: "Some make", Model: "Some model", Vin: &vin, Currency: "GBP",
		CurrentOdometer: 4200, TeslaMateAuthType: models.AuthModeNone}
	theirs := &models.Vehicle{UserID: other.ID, Name: "Not mine", Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
	for _, v := range []*models.Vehicle{mine, theirs} {
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
	}

	h := NewIntegrationReadHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	router := chi.NewRouter()
	router.Get("/vehicles", h.ListVehicles)
	router.Get("/vehicles/{vehicleId}", h.GetVehicle)
	router.Get("/vehicles/{vehicleId}/costs", h.GetCosts)
	router.Get("/vehicles/{vehicleId}/data-gaps", h.GetDataGaps)
	router.Get("/vehicles/{vehicleId}/trips", h.GetTrips)
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, owner.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for _, path := range []string{"/vehicles", "/vehicles/" + mine.ID, "/vehicles/" + mine.ID + "/costs", "/vehicles/" + mine.ID + "/data-gaps", "/vehicles/" + mine.ID + "/trips?search=lyon&limit=5"} {
		rec := get(path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), vin) || strings.Contains(rec.Body.String(), theirs.ID) {
			t.Errorf("%s leaked the VIN or another account's vehicle: %s", path, rec.Body.String())
		}
	}
	for _, path := range []string{"/vehicles/" + theirs.ID, "/vehicles/" + theirs.ID + "/costs", "/vehicles/" + theirs.ID + "/data-gaps", "/vehicles/" + theirs.ID + "/trips"} {
		if rec := get(path); rec.Code != http.StatusNotFound {
			t.Errorf("%s: another account's vehicle must be a 404, got %d", path, rec.Code)
		}
	}

	for _, bad := range []string{"limit=0", "limit=abc", "from=yesterday"} {
		if rec := get("/vehicles/" + mine.ID + "/trips?" + bad); rec.Code != http.StatusBadRequest {
			t.Errorf("trips?%s: got %d, want 400", bad, rec.Code)
		}
	}

	var one IntegrationVehicleV1
	if err := json.Unmarshal(get("/vehicles/"+mine.ID).Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.Odometer != (IntegrationDistance{Value: 4200, Unit: "km"}) || one.Currency != "GBP" {
		t.Errorf("vehicle: %+v", one)
	}

	var costs IntegrationCosts
	if err := json.Unmarshal(get("/vehicles/"+mine.ID+"/costs").Body.Bytes(), &costs); err != nil {
		t.Fatal(err)
	}
	if costs.Currency != "GBP" || costs.SincePurchase.Costs.Total.Currency != "GBP" || costs.Last12Months.Costs.Energy.Currency != "GBP" {
		t.Errorf("every amount carries the vehicle's currency: %+v", costs)
	}
}
