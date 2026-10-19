package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func monthStart(s string) *time.Time {
	t, _ := time.Parse("2006-01", s)
	return &t
}

func monthlySummary(n int) *services.TCOSummary {
	last := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	months := monthsUpTo(last, n)
	months[len(months)-1].Scheduled = 500
	months[0].DistanceKm = 0
	months[0].Total = 0
	return &services.TCOSummary{MonthlyCosts: months, CurrentMonth: "2026-10"}
}

func monthsOf(out IntegrationMonthlyCosts) string {
	names := make([]string, len(out.Months))
	for i, m := range out.Months {
		names[i] = m.Month
	}
	return strings.Join(names, ",")
}

func TestIntegrationMonthlyCostsDefaultToTheLastTwelveMonthsAndFlagTheMonthInProgress(t *testing.T) {
	out := buildIntegrationMonthlyCosts("v1", "CHF", monthlySummary(30), nil, nil)

	if len(out.Months) != 12 || out.Months[0].Month != "2025-11" || out.Months[11].Month != "2026-10" {
		t.Fatalf("twelve months ending with the month in progress: %s", monthsOf(out))
	}
	last := out.Months[11]
	if !last.InProgress || out.Months[10].InProgress {
		t.Errorf("only the month in progress is flagged: %+v / %+v", last.InProgress, out.Months[10].InProgress)
	}
	if last.Scheduled == nil || *last.Scheduled != (IntegrationAmount{Amount: "5.00", Currency: "CHF"}) || out.Months[10].Scheduled != nil {
		t.Errorf("instalments due later in the month in progress are named, a past month has none: %+v", last.Scheduled)
	}
	if last.Costs.Total != (IntegrationAmount{Amount: "37.00", Currency: "CHF"}) || last.CostPerKm == nil || last.CostPerKm.Amount != "0.3700" {
		t.Errorf("a month's total and rate: %+v %+v", last.Costs.Total, last.CostPerKm)
	}
	if out.Total.Total != (IntegrationAmount{Amount: "444.00", Currency: "CHF"}) || out.Distance.Value != 1200 || out.CostPerKm.Amount != "0.3700" {
		t.Errorf("the total adds up the listed months only: %+v %+v %+v", out.Total.Total, out.Distance, out.CostPerKm)
	}
}

func TestIntegrationMonthlyCostsHonourTheBoundsAndTheCap(t *testing.T) {
	sum := monthlySummary(30)

	between := buildIntegrationMonthlyCosts("v1", "EUR", sum, monthStart("2026-03"), monthStart("2026-05"))
	if monthsOf(between) != "2026-03,2026-04,2026-05" {
		t.Errorf("both bounds are included: %s", monthsOf(between))
	}
	if from := buildIntegrationMonthlyCosts("v1", "EUR", sum, monthStart("2026-09"), nil); monthsOf(from) != "2026-09,2026-10" {
		t.Errorf("from alone runs to the latest month: %s", monthsOf(from))
	}
	if upTo := buildIntegrationMonthlyCosts("v1", "EUR", sum, nil, monthStart("2024-06")); len(upTo.Months) == 0 || upTo.Months[len(upTo.Months)-1].Month != "2024-06" {
		t.Errorf("to alone ends at that month: %s", monthsOf(upTo))
	}

	all := monthlySummary(200).MonthlyCosts
	long := buildIntegrationMonthlyCosts("v1", "EUR", monthlySummary(200), monthStart("2010-01"), nil)
	if len(long.Months) != integrationMonthsMax || long.Months[0].Month != all[0].Month {
		t.Errorf("a request with a start is capped at %d months after it: got %d from %s", integrationMonthsMax, len(long.Months), long.Months[0].Month)
	}
	recent := buildIntegrationMonthlyCosts("v1", "EUR", monthlySummary(200), nil, monthStart("2026-10"))
	if len(recent.Months) != integrationMonthsMax || recent.Months[len(recent.Months)-1].Month != "2026-10" {
		t.Errorf("a request with an end keeps the months nearest it: got %d ending %s", len(recent.Months), recent.Months[len(recent.Months)-1].Month)
	}

	none := buildIntegrationMonthlyCosts("v1", "EUR", sum, monthStart("2030-01"), nil)
	body, _ := json.Marshal(none)
	if !strings.Contains(string(body), `"months":[]`) || none.CostPerKm != nil {
		t.Errorf("a period with no month is an empty list and an unknown rate: %s", body)
	}
}

func TestIntegrationMonthlyCostsRouteRefusesAMalformedMonthAndAForeignVehicle(t *testing.T) {
	repo := authTestRepo(t)
	ctx := t.Context()
	owner, _ := repo.CreateUser(ctx, "monthly-owner@example.com", "hash")
	other, _ := repo.CreateUser(ctx, "monthly-other@example.com", "hash")
	mine := &models.Vehicle{UserID: owner.ID, Name: "Daily", Currency: "GBP", TeslaMateAuthType: models.AuthModeNone}
	theirs := &models.Vehicle{UserID: other.ID, Name: "Not mine", Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
	for _, v := range []*models.Vehicle{mine, theirs} {
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	h := NewIntegrationReadHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	router := chi.NewRouter()
	router.Get("/vehicles/{vehicleId}/monthly-costs", h.GetMonthlyCosts)
	get := func(vehicleID, query string) (int, string) {
		req := httptest.NewRequest(http.MethodGet, "/vehicles/"+vehicleID+"/monthly-costs"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, owner.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	if code, body := get(mine.ID, ""); code != http.StatusOK || !strings.Contains(body, `"currency":"GBP"`) || strings.Contains(body, theirs.ID) {
		t.Errorf("own vehicle: %d %s", code, body)
	}
	if code, _ := get(mine.ID, "?from=2026-13"); code != http.StatusBadRequest {
		t.Errorf("a malformed month: status %d, want 400", code)
	}
	if code, _ := get(mine.ID, "?from=2026-05&to=2026-01"); code != http.StatusBadRequest {
		t.Errorf("a start after the end: status %d, want 400", code)
	}
	if code, _ := get(theirs.ID, ""); code != http.StatusNotFound {
		t.Errorf("another account's vehicle: status %d, want 404", code)
	}
}
