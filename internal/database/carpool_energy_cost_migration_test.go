package database

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/migrations"
)

// The energy of a carpool is stored under neutral names: the migration keeps every amount, moves the fuel of a vehicle
// that burns fuel only into fuel_cost, leaves electricity_cost as the electric part and renames the overrides.
func TestCarpoolEnergyCostMigrationKeepsAmountsAndSplitsFuel(t *testing.T) {
	repo, ctx, owner, _, electric := vehicleRepoFixture(t, "carpool-energy-migration")
	ice := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: "ICE", Powertrain: models.PowertrainICE}
	if err := repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	save := func(v *models.Vehicle, title string, energy, fuel int64) string {
		trip := &models.CarpoolTrip{VehicleID: v.ID, Title: title, Date: now}
		leg := models.CarpoolLeg{DistanceKm: 10, EnergyCost: money.Cents(energy), FuelCost: money.Cents(fuel), Overrides: map[string]money.Cents{models.CarpoolCostEnergy: money.Cents(energy)}}
		if err := repo.CreateCarpoolTrip(ctx, trip, []models.CarpoolLeg{leg}, nil); err != nil {
			t.Fatal(err)
		}
		return trip.ID
	}
	evID := save(electric, "ev", 500, 0)
	iceID := save(ice, "ice", 960, 960)

	// Back to the layout before the migration (one electricity_cost column, the fuel in it), then forward again.
	for _, name := range []string{"000081_carpool_energy_cost.down.sql", "000081_carpool_energy_cost.up.sql"} {
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	check := func(vehicleID, tripID string, energy, fuel, electricity int64) {
		t.Helper()
		got, err := repo.GetCarpoolTrip(ctx, tripID, vehicleID)
		if err != nil {
			t.Fatal(err)
		}
		if got.EnergyCost != money.Cents(energy) || got.FuelCost != money.Cents(fuel) || got.ElectricityCost != money.Cents(electricity) {
			t.Errorf("trip: energy %v fuel %v electricity %v, want %d / %d / %d", got.EnergyCost, got.FuelCost, got.ElectricityCost, energy, fuel, electricity)
		}
		legs := got.Legs
		if len(legs) != 1 || legs[0].EnergyCost != money.Cents(energy) || legs[0].FuelCost != money.Cents(fuel) || legs[0].ElectricityCost != money.Cents(electricity) {
			t.Errorf("leg: %+v", legs)
		}
		if legs[0].Overrides[models.CarpoolCostEnergy] != money.Cents(energy) || len(legs[0].Overrides) != 1 {
			t.Errorf("the override moves to the neutral name: %+v", legs[0].Overrides)
		}
	}
	check(electric.ID, evID, 500, 0, 500)
	check(ice.ID, iceID, 960, 960, 0)
}
