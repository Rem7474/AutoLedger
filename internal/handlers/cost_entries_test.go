package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// The lines of a month add up to what the monthly cost of that month says was paid, and what the passengers paid
// is the carpool revenue of the same month: the screens that list the lines and the ones that total them agree.
func TestCostEntriesAddUpToTheMonthlyCost(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "entries-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := repo.CreateUser(ctx, "entries-stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := repo.CreateUser(ctx, "entries-viewer@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: "EV", Powertrain: models.PowertrainEV, TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddVehicleMember(ctx, v.ID, viewer.Email, models.RoleViewer, nil); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO drive_expenses (vehicle_id, type, amount, date) VALUES ($1, 'TOLL', 12.5, '2026-03-10T10:00:00Z')`,
		`INSERT INTO maintenance_expenses (vehicle_id, category, amount, date, description) VALUES ($1, 'MAINTENANCE', 80, '2026-03-12T10:00:00Z', 'Brakes')`,
		`INSERT INTO maintenance_expenses (vehicle_id, category, amount, date, description) VALUES ($1, 'INSURANCE', 30, '2026-03-31T22:30:00Z', 'Insurance')`,
		`INSERT INTO maintenance_expenses (vehicle_id, category, amount, date, description) VALUES ($1, 'TAX', 5, '2026-04-02T10:00:00Z', 'April')`,
		`INSERT INTO carpool_trips (vehicle_id, title, date, total_revenue) VALUES ($1, 'Commute', '2026-03-15T10:00:00Z', 20)`,
		`INSERT INTO carpool_trips (vehicle_id, title, date, total_revenue) VALUES ($1, 'Next month', '2026-04-03T10:00:00Z', 7)`,
	} {
		if _, err := repo.Pool().Exec(ctx, q, v.ID); err != nil {
			t.Fatal(err)
		}
	}

	tcoService := services.NewTCOService(repo.Pool(), "Europe/Paris")
	h := NewTCOHandler(repo, tcoService)
	router := chi.NewRouter()
	router.Get("/vehicles/{vehicleId}/cost-entries", h.GetCostEntries)
	call := func(userID, query string) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/vehicles/"+v.ID+"/cost-entries"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}
	amount := func(body map[string]any, key string) float64 {
		a, _ := body[key].(map[string]any)
		f, _ := strconv.ParseFloat(a["amount"].(string), 64)
		return f
	}

	status, body := call(owner.ID, "?from=2026-03-01&to=2026-03-31")
	if status != http.StatusOK {
		t.Fatalf("owner: %d %v", status, body)
	}
	// The insurance paid at 23:30 on the 31st in Paris is dated April 1st there: the month follows the timezone.
	entries := body["entries"].([]any)
	if len(entries) != 2 || amount(body, "paid_total") != 92.5 || amount(body, "carpool_received") != 20 || amount(body, "net_total") != 72.5 {
		t.Fatalf("march: %v", body)
	}
	first := entries[0].(map[string]any)
	if first["source"] == "" || first["source_id"] == "" {
		t.Fatalf("a line names the record it comes from: %v", first)
	}

	summary, err := tcoService.ComputeVehicleTCO(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	var march *services.MonthlyCost
	for i := range summary.MonthlyCosts {
		if summary.MonthlyCosts[i].Month == "2026-03" {
			march = &summary.MonthlyCosts[i]
		}
	}
	if march == nil {
		t.Fatal("no monthly cost for 2026-03")
	}
	paid := march.Total - march.SmoothedEnergy - march.Scheduled
	if got := amount(body, "paid_total"); got != paid.Float() || amount(body, "carpool_received") != march.CarpoolRevenue.Float() ||
		amount(body, "net_total") != (paid-march.CarpoolRevenue).Float() {
		t.Fatalf("the lines of March (%v) differ from the monthly cost %+v", body, *march)
	}

	// A category filter narrows the lines and drops the carpool figure, which belongs to the whole month.
	_, body = call(owner.ID, "?from=2026-03-01&to=2026-03-31&category=toll&limit=1")
	if amount(body, "paid_total") != 12.5 || amount(body, "carpool_received") != 0 || len(body["entries"].([]any)) != 1 {
		t.Fatalf("toll: %v", body)
	}
	_, body = call(owner.ID, "?limit=1")
	if body["next_offset"] != float64(1) {
		t.Fatalf("a cut page gives the offset of the next one: %v", body["next_offset"])
	}

	if status, _ := call(viewer.ID, ""); status != http.StatusOK {
		t.Errorf("a member can read the lines, got %d", status)
	}
	if status, _ := call(stranger.ID, ""); status != http.StatusNotFound {
		t.Errorf("another user's vehicle must answer 404, got %d", status)
	}
	if status, _ := call(owner.ID, "?from=nope"); status != http.StatusBadRequest {
		t.Errorf("a bad date must answer 400, got %d", status)
	}
	if status, _ := call(owner.ID, "?category=nope"); status != http.StatusBadRequest {
		t.Errorf("an unknown category must answer 400, got %d", status)
	}
}
