package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// limitedToken returns a read token of the owner that only reaches the given vehicles.
func (f *mcpFixture) limitedToken(t *testing.T, vehicleIDs ...string) string {
	t.Helper()
	raw, prefix, hash, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.CreateAPIToken(context.Background(), f.owner.ID, "limited", auth.ScopeRead, hash, prefix, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.Pool().Exec(context.Background(), `UPDATE api_tokens SET vehicle_ids = $1::uuid[] WHERE token_hash = $2;`, vehicleIDs, hash); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestMCPTokenLimitedToSomeVehiclesReachesOnlyThose(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	second := &models.Vehicle{UserID: f.owner.ID, Name: "Van", Currency: "GBP", TeslaMateAuthType: models.AuthModeNone}
	if err := f.repo.CreateVehicle(ctx, second); err != nil {
		t.Fatal(err)
	}
	session := f.connect(t, f.limitedToken(t, f.mine.ID))

	names := func(res *mcp.CallToolResult) string {
		raw, _ := json.Marshal(res.StructuredContent)
		return string(raw)
	}
	list, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_vehicles"})
	if err != nil || list.IsError {
		t.Fatalf("list_vehicles: %v %+v", err, list)
	}
	if got := names(list); !strings.Contains(got, f.mine.ID) || strings.Contains(got, second.ID) || strings.Contains(got, "Van") {
		t.Errorf("the list holds the granted vehicle only: %s", got)
	}

	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_vehicle_costs", Arguments: map[string]any{"vehicle_id": f.mine.ID}}); err != nil || res.IsError {
		t.Errorf("the granted vehicle is readable: %v %+v", err, res)
	}
	for _, ref := range []string{second.ID, "Van"} {
		if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_vehicle_costs", Arguments: map[string]any{"vehicle_id": ref}}); err != nil || !res.IsError {
			t.Errorf("%q is outside the grant and refused: %v %+v", ref, err, res)
		}
	}

	fleet, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_fleet_summary"})
	if err != nil || !fleet.IsError || !strings.Contains(fleet.Content[0].(*mcp.TextContent).Text, "limited to some vehicles") {
		t.Errorf("the household view adds up every vehicle, so a limited grant is refused with a way forward: %v %+v", err, fleet)
	}

	account, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "autoledger://account"})
	if err != nil || strings.Contains(account.Contents[0].Text, second.ID) || !strings.Contains(account.Contents[0].Text, f.mine.ID) {
		t.Errorf("the account resource lists the granted vehicles only: %v %+v", err, account)
	}
	if _, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "autoledger://vehicles/" + second.ID}); err == nil {
		t.Error("a vehicle resource outside the grant is not found")
	}

	// A scenario tracking a vehicle outside the grant is hidden, whatever the way to ask for it.
	hidden := &models.ComparisonScenario{UserID: f.owner.ID, Name: "Van against diesel", Currency: "GBP", Mode: models.ComparisonModeRetrospective, VehicleID: &second.ID, AnnualKm: 15000, Years: 5,
		ICE: models.ICEInputs{FuelType: "DIESEL", LPer100Km: 6, FuelPrice: 1.7, PurchasePrice: 2500000, ResaleValue: 1000000}}
	if err := f.repo.CreateComparisonScenario(ctx, hidden); err != nil {
		t.Fatal(err)
	}
	for _, args := range []map[string]any{{}, {"scenario": hidden.ID}} {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_comparison", Arguments: args})
		if err != nil || strings.Contains(names(res)+textOf(res), "Van against diesel") {
			t.Errorf("the scenario of a vehicle outside the grant stays hidden: %v %+v", err, res)
		}
	}

	// An unlimited token of the same account still sees both.
	all := f.connect(t, f.readTok)
	both, err := all.CallTool(ctx, &mcp.CallToolParams{Name: "list_vehicles"})
	if err != nil || !strings.Contains(names(both), second.ID) || !strings.Contains(names(both), f.mine.ID) {
		t.Errorf("an unlimited token sees every vehicle: %v %s", err, names(both))
	}
}

func textOf(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	return b.String()
}

func TestVehicleScopeAlsoLimitsTheHomeAssistantViewsAndTheSharedVehicleCheck(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "scope-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, name := range []string{"Daily", "Van"} {
		v := &models.Vehicle{UserID: owner.ID, Name: name, Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.ID)
	}
	h := NewHomeAssistantHandler(repo, nil)
	router := chi.NewRouter()
	router.Get("/vehicles", h.ListVehicles)
	router.Get("/vehicles/{vehicleId}", h.GetVehicle)
	router.Get("/vehicles/{vehicleId}/metrics", h.GetVehicleMetrics)
	get := func(path string, limitedTo ...string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		c := context.WithValue(req.Context(), middleware.UserIDKey, owner.ID)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(middleware.WithVehicleScope(c, limitedTo)))
		return rec
	}

	if body := get("/vehicles", ids[0]).Body.String(); !strings.Contains(body, ids[0]) || strings.Contains(body, ids[1]) {
		t.Errorf("the vehicle list holds the granted vehicle only: %s", body)
	}
	if body := get("/vehicles").Body.String(); !strings.Contains(body, ids[0]) || !strings.Contains(body, ids[1]) {
		t.Errorf("without a limit both are listed: %s", body)
	}
	for _, path := range []string{"/vehicles/" + ids[1], "/vehicles/" + ids[1] + "/metrics"} {
		if rec := get(path, ids[0]); rec.Code != http.StatusNotFound {
			t.Errorf("%s is outside the grant: %d", path, rec.Code)
		}
		if rec := get(path); rec.Code != http.StatusOK {
			t.Errorf("%s without a limit: %d %s", path, rec.Code, rec.Body.String())
		}
	}
}
