package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestParseSummaryIncludeAcceptsTheSectionsAndRefusesTheRest(t *testing.T) {
	got, err := parseSummaryInclude(" Tires, battery ,,sources")
	if err != nil || !got.Tires || !got.Battery || !got.Sources || got.Ownership {
		t.Errorf("names are trimmed, case-insensitive and empty parts ignored: %+v %v", got, err)
	}
	if none, err := parseSummaryInclude(""); err != nil || none != (SummaryInclude{}) {
		t.Errorf("no include asks for no section: %+v %v", none, err)
	}
	if _, err := parseSummaryInclude("tires,tyres"); err == nil || err.Code != "request.invalid_value" || !strings.Contains(err.Message, "ownership") {
		t.Errorf("a typo is refused with the list of sections, not silently dropped: %+v", err)
	}
}

func TestIntegrationSummaryWithoutIncludeCarriesNoOptionalSection(t *testing.T) {
	f := newWhoAmIFixture(t)
	_, body := readSummary(t, f, "homeassistant", nil)
	for _, key := range []string{`"worst_wear_pct"`, `"latest_percent"`, `"depreciation_known"`, `"teslamate_connected"`} {
		if strings.Contains(body, key) {
			t.Errorf("%s is opt-in: %s", key, body)
		}
	}
}

func TestIntegrationSummaryRefusesAnUnknownSection(t *testing.T) {
	f := newWhoAmIFixture(t)
	rec := f.get("/summary?include=tires,wheels", "homeassistant", nil)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "request.invalid_value") {
		t.Errorf("an unknown section is a 400 with a stable code: %d %s", rec.Code, rec.Body.String())
	}
}

func TestIntegrationSummaryIncludesSourcesOwnershipAndBatteryWhenAsked(t *testing.T) {
	f := newWhoAmIFixture(t)
	ctx := context.Background()
	h := NewIntegrationReadHandler(f.repo, services.NewTCOService(f.repo.Pool(), "UTC")).WithEnergy(services.NewEnergyStatsService(f.repo.Pool(), "UTC"))

	health, capacity := 91.5, 70.0
	if err := f.repo.UpsertBatterySnapshot(ctx, f.daily.ID, time.Now().UTC().AddDate(0, 0, -2), models.BatterySnapshot{HealthPercent: &health, CurrentCapacityKwh: &capacity}); err != nil {
		t.Fatal(err)
	}
	out, err := h.Summary(ctxOf(f), f.owner.ID, SummaryInclude{Battery: true, Ownership: true, Sources: true})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]IntegrationSummaryVehicle{}
	for _, v := range out.Vehicles {
		byName[v.Name] = v
	}
	daily, van := byName["Daily"], byName["Van"]

	if daily.Battery == nil || daily.Battery.LatestPercent != 91.5 || daily.Battery.CapacityKwh == nil || *daily.Battery.CapacityKwh != 70 {
		t.Errorf("the battery reading is carried: %+v", daily.Battery)
	}
	if van.Battery != nil {
		t.Errorf("a vehicle without a reading has no battery section, not an empty one: %+v", van.Battery)
	}
	if daily.Tires != nil || van.Tires != nil {
		t.Errorf("tires were not asked for: %+v %+v", daily.Tires, van.Tires)
	}
	if daily.Ownership == nil || daily.Ownership.CurrentValue != nil || daily.Ownership.Mileage != nil {
		t.Errorf("a vehicle bought outright, without a resale service, has a cost but no value or allowance: %+v", daily.Ownership)
	}
	if daily.Sources == nil || daily.Sources.PendingCharges != 1 || daily.Sources.TeslaMateConnected || daily.Sources.LastSync != nil {
		t.Errorf("the pushed charge waiting for a vehicle is counted: %+v", daily.Sources)
	}
	if van.Sources == nil || van.Sources.PendingCharges != 1 {
		t.Errorf("the pending count is the account's, on every vehicle: %+v", van.Sources)
	}
}

func TestIntegrationSummaryTiresAreTheWorstOfTheMountedOnes(t *testing.T) {
	f := newWhoAmIFixture(t)
	ctx := context.Background()
	h := NewIntegrationReadHandler(f.repo, services.NewTCOService(f.repo.Pool(), "UTC")).
		WithEnergy(services.NewEnergyStatsService(f.repo.Pool(), "UTC")).WithTires(services.NewTireWearService(f.repo))

	mounted := 0.0
	add := func(pos models.TirePosition, lifespan int, driven float64) {
		t.Helper()
		if err := f.repo.CreateTire(ctx, &models.Tire{VehicleID: &f.daily.ID, Brand: "B", Model: "M", Dimension: "205", Season: models.TireSeasonSummer,
			PurchaseDate: time.Now().AddDate(-1, 0, 0), PurchasePrice: 20000, CurrentPosition: pos, InitialDepthMm: 8, MinLegalDepthMm: 1.6,
			MountedOdometer: &mounted, EstimatedLifespanKm: lifespan, AccumulatedDistanceKm: driven}); err != nil {
			t.Fatal(err)
		}
	}
	add(models.TirePosFL, 40000, 0)
	add(models.TirePosFR, 25000, 0)
	add(models.TirePosStorage, 5000, 30000) // far more worn than the mounted ones, and not on the car

	out, err := h.Summary(ctxOf(f), f.owner.ID, SummaryInclude{Tires: true})
	if err != nil {
		t.Fatal(err)
	}
	var daily, van IntegrationSummaryVehicle
	for _, v := range out.Vehicles {
		switch v.Name {
		case "Daily":
			daily = v
		case "Van":
			van = v
		}
	}
	if van.Tires != nil {
		t.Errorf("a vehicle without tire has no tires section: %+v", van.Tires)
	}
	if daily.Tires == nil || daily.Tires.Mounted != 2 {
		t.Fatalf("two tires are mounted, the stored one is left out: %+v", daily.Tires)
	}

	// The figures are the worst of the per-vehicle tires route, restricted to the mounted tires.
	route, err := h.Tires(context.Background(), f.owner.ID, f.daily.ID)
	if err != nil {
		t.Fatal(err)
	}
	var worstLife, worstWear float64
	stored := 0.0
	for _, tire := range route.Tires {
		if tire.Position == "STORAGE" {
			stored = tire.LifeUsedPct
			continue
		}
		worstLife, worstWear = max(worstLife, tire.LifeUsedPct), max(worstWear, tire.WearPct)
	}
	if daily.Tires.WorstLifeUsedPct != worstLife || daily.Tires.WorstWearPct != worstWear {
		t.Errorf("worst figures of the mounted tires: %+v, want life %v wear %v", daily.Tires, worstLife, worstWear)
	}
	if stored <= worstLife {
		t.Fatalf("the fixture must make the stored tire the most worn one (%v vs %v)", stored, worstLife)
	}
	if daily.Tires.WorstCondition == "" {
		t.Errorf("the worst condition is named: %+v", daily.Tires)
	}
}

func TestSummaryTiresKeepsTheSmallestRemainingDistanceAndTheNearestDate(t *testing.T) {
	d := func(day string) *services.TireForecast { return &services.TireForecast{ReplacementDate: day} }
	mk := func(pos models.TirePosition, condition string, remaining float64, date string) services.TireWearStats {
		return services.TireWearStats{
			Tire: models.Tire{CurrentPosition: pos, PurchaseDate: time.Now()}, Condition: condition, WearPercentage: 10, LifeProgressPct: 20,
			DynamicRemainingKm: remaining, DrivesCount: 1, ReplacementForecast: d(date),
		}
	}
	got := summaryTires("v", "EUR", []services.TireWearStats{
		mk(models.TirePosFL, "GOOD", 30000, "2028-02-01"),
		mk(models.TirePosFR, "CRITICAL", 4000, "2027-03-01"),
		mk(models.TirePosRL, "WARNING", 9000, ""),
		mk(models.TirePosDisposed, "CRITICAL", 1, "2020-01-01"),
	})
	if got == nil || got.Mounted != 3 || got.WorstCondition != "CRITICAL" || got.SmallestRemaining == nil || got.SmallestRemaining.Value != 4000 ||
		got.NearestReplacementDate == nil || *got.NearestReplacementDate != "2027-03-01" {
		t.Errorf("the worst of three mounted tires, the disposed one ignored: %+v", got)
	}
	if summaryTires("v", "EUR", []services.TireWearStats{mk(models.TirePosStorage, "GOOD", 1, "")}) != nil {
		t.Error("only stored tires: nothing is mounted, so no section")
	}
}

func TestSummarySourcesKeepsTheLatestInstantOfEachKind(t *testing.T) {
	early, late := "2026-11-01T10:00:00Z", "2026-11-04T08:00:00Z"
	got := summarySources(IntegrationDataSources{
		TeslaMate: true, PendingCharges: 2,
		Sources: []IntegrationSource{{Origin: "manual", LastRecord: &early}, {Origin: "webhook", LastRecord: &late}, {Origin: "csv"}},
		Sync:    []IntegrationSyncResource{{Resource: "drives", LastSuccess: &early}, {Resource: "charges", LastSuccess: &late}, {Resource: "x"}},
	})
	if !got.TeslaMateConnected || got.PendingCharges != 2 || got.LastRecord == nil || *got.LastRecord != late || got.LastSync == nil || *got.LastSync != late {
		t.Errorf("latest record and latest sync: %+v", got)
	}
	empty := summarySources(IntegrationDataSources{})
	if empty.LastRecord != nil || empty.LastSync != nil {
		t.Errorf("no record and no sync are null: %+v", empty)
	}
}

func TestSummaryOwnershipCarriesTheValueAndTheLeaseAllowance(t *testing.T) {
	perKm := &IntegrationAmount{Amount: "0.4200", Currency: "EUR"}
	mileage := &IntegrationMileage{Driven: kilometres(12000)}
	got := summaryOwnership(IntegrationOwnership{
		Economic: IntegrationEconomicCost{FullCostPerKm: perKm, DepreciationKnown: true},
		Resale:   &IntegrationResale{CurrentValue: IntegrationAmount{Amount: "18000.00", Currency: "EUR"}},
		Contract: &IntegrationContract{Mileage: mileage},
	})
	if got.FullCostPerKm != perKm || !got.DepreciationKnown || got.CurrentValue == nil || got.CurrentValue.Amount != "18000.00" || got.Mileage != mileage {
		t.Errorf("value, cost per km and allowance: %+v", got)
	}
	bare := summaryOwnership(IntegrationOwnership{})
	if bare.CurrentValue != nil || bare.Mileage != nil || bare.FullCostPerKm != nil || bare.DepreciationKnown {
		t.Errorf("nothing known stays null and false: %+v", bare)
	}
}

// ctxOf is a request context for the owner of the fixture with no restriction on the vehicles.
func ctxOf(f *whoAmIFixture) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, f.owner.ID)
}
