package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func callStructured[T any](t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) (out T, isError bool) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if res.IsError {
		return out, true
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v (%s)", tool, err, raw)
	}
	return out, false
}

func TestMCPTariffPlansAreListedAndAChargeIsPricedWithoutStoringAnything(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	session := f.connect(t, f.readTok)
	otherSession := f.connect(t, f.otherTok)

	empty, isErr := callStructured[IntegrationTariffPlans](t, session, "list_tariff_plans", nil)
	if isErr || empty.Plans == nil || len(empty.Plans) != 0 {
		t.Fatalf("an account without a plan lists none: %+v", empty)
	}
	// With no plan at all a charge is not priced, and says why.
	none, isErr := callStructured[IntegrationChargeEstimate](t, otherSession, "estimate_charge_cost", map[string]any{"energy_kwh": 10, "duration_minutes": 60})
	if isErr || none.Priced || none.Cost != nil || none.Plan != nil || none.UnpricedReason == nil || *none.UnpricedReason == "" || none.Bands == nil {
		t.Errorf("no plan, no price: %+v", none)
	}

	night := &models.TariffPlan{UserID: f.owner.ID, Name: "Night", PlanType: "BANDS", Currency: "GBP", DefaultBand: "day", IsDefault: true,
		Bands: []models.TariffBand{{Name: "day", RateCents: 200000}, {Name: "night", RateCents: 100000}},
		Rules: []models.TariffRule{{Start: "22:00", End: "06:00", Band: "night"}}}
	if err := f.repo.CreateTariffPlan(ctx, night); err != nil {
		t.Fatal(err)
	}
	expired := "2020-01-01"
	old := &models.TariffPlan{UserID: f.owner.ID, Name: "Old", PlanType: "FLAT", Currency: "GBP", FlatRateCents: ptr(money.Rate(200000)), ValidTo: &expired}
	theirs := &models.TariffPlan{UserID: f.theirs.UserID, Name: "Theirs", PlanType: "FLAT", Currency: "EUR", FlatRateCents: ptr(money.Rate(200000))}
	for _, p := range []*models.TariffPlan{old, theirs} {
		if err := f.repo.CreateTariffPlan(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	plans, isErr := callStructured[IntegrationTariffPlans](t, session, "list_tariff_plans", nil)
	if isErr || len(plans.Plans) != 2 || plans.Plans[0].ID != night.ID || !plans.Plans[0].IsDefault {
		t.Fatalf("the account's own plans, the default first: %+v", plans)
	}
	got := plans.Plans[0]
	if len(got.Bands) != 2 || got.Bands[0].Name != "day" || got.Bands[0].PricePerKwh != "0.20" || got.Bands[1].PricePerKwh != "0.10" ||
		len(got.Rules) != 1 || got.Rules[0].Band != "night" || got.Rules[0].Days == nil || got.DefaultBand != "day" {
		t.Errorf("bands and rules of the plan: %+v", got)
	}
	if plans.Plans[1].Name != "Old" || plans.Plans[1].ValidTo == nil || *plans.Plans[1].ValidTo != expired || len(plans.Plans[1].Rules) != 0 || len(plans.Plans[1].Bands) != 1 || plans.Plans[1].Bands[0].Name != "FLAT" {
		t.Errorf("a flat plan has one band and no rules: %+v", plans.Plans[1])
	}

	// 20:00-04:00 UTC, 10 kWh: 2 of the 8 hours are day, 6 are night.
	charge := map[string]any{"energy_kwh": 10, "duration_minutes": 480, "start": "2026-07-10T20:00:00Z"}
	byDefault, isErr := callStructured[IntegrationChargeEstimate](t, session, "estimate_charge_cost", charge)
	if isErr || !byDefault.Priced || byDefault.Cost == nil || byDefault.Cost.Amount != "1.25" || byDefault.Cost.Currency != "GBP" || byDefault.Plan == nil || byDefault.Plan.Name != "Night" ||
		byDefault.AveragePricePerKwh == nil || *byDefault.AveragePricePerKwh != "0.125" || byDefault.UnpricedReason != nil {
		t.Fatalf("the default plan prices the charge: %+v", byDefault)
	}
	if len(byDefault.Bands) != 2 || byDefault.Bands[0].Band != "night" || byDefault.Bands[0].Minutes != 360 || byDefault.Bands[0].Kwh != 7.5 || byDefault.Bands[0].PricePerKwh != "0.10" ||
		byDefault.Bands[1].Band != "day" || byDefault.Bands[1].Minutes != 120 || byDefault.Bands[1].Kwh != 2.5 {
		t.Errorf("the charge splits between the bands, the largest share first: %+v", byDefault.Bands)
	}

	// A vehicle without a plan of its own falls back to the default plan; one with a plan uses it.
	byVehicle, isErr := callStructured[IntegrationChargeEstimate](t, session, "estimate_charge_cost", map[string]any{"energy_kwh": 10, "duration_minutes": 480, "start": "2026-07-10T20:00:00Z", "vehicle_id": f.mine.ID})
	if isErr || !byVehicle.Priced || byVehicle.Cost.Amount != "1.25" {
		t.Errorf("a vehicle prices with the default plan: %+v", byVehicle)
	}
	for k, v := range map[string]any{"plan_id": night.ID} {
		charge[k] = v
	}
	byPlan, isErr := callStructured[IntegrationChargeEstimate](t, session, "estimate_charge_cost", charge)
	if isErr || !byPlan.Priced || byPlan.Cost.Amount != "1.25" {
		t.Errorf("a plan chosen by id prices the charge: %+v", byPlan)
	}

	late, isErr := callStructured[IntegrationChargeEstimate](t, session, "estimate_charge_cost", map[string]any{"energy_kwh": 10, "duration_minutes": 60, "start": "2026-07-10T12:00:00Z", "plan_id": old.ID})
	if isErr || late.Priced || late.UnpricedReason == nil || !strings.Contains(*late.UnpricedReason, "does not apply") || late.Cost != nil {
		t.Errorf("a plan out of its validity range prices nothing and says so: %+v", late)
	}

	// Nothing the estimate does is stored.
	var charges int
	if err := f.repo.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM charge_logs;`).Scan(&charges); err != nil || charges != 0 {
		t.Errorf("an estimate stores nothing: %d %v", charges, err)
	}

	for name, args := range map[string]map[string]any{
		"no energy":          {"energy_kwh": 0, "duration_minutes": 60},
		"too much energy":    {"energy_kwh": 5000, "duration_minutes": 60},
		"no duration":        {"energy_kwh": 10, "duration_minutes": 0},
		"endless duration":   {"energy_kwh": 10, "duration_minutes": 100000},
		"start without zone": {"energy_kwh": 10, "duration_minutes": 60, "start": "2026-07-10T20:00:00"},
		"vehicle and plan":   {"energy_kwh": 10, "duration_minutes": 60, "vehicle_id": f.mine.ID, "plan_id": night.ID},
		"unknown plan":       {"energy_kwh": 10, "duration_minutes": 60, "plan_id": "00000000-0000-0000-0000-000000000000"},
		"another's plan":     {"energy_kwh": 10, "duration_minutes": 60, "plan_id": theirs.ID},
		"another's vehicle":  {"energy_kwh": 10, "duration_minutes": 60, "vehicle_id": f.theirs.ID},
	} {
		if _, isErr := callStructured[IntegrationChargeEstimate](t, session, "estimate_charge_cost", args); !isErr {
			t.Errorf("%s is refused", name)
		}
	}
}
