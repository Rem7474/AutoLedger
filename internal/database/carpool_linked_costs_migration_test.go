package database

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/migrations"
)

func TestCarpoolLinkedCostsMigrationFreezesCarpoolsMadeFromDrives(t *testing.T) {
	repo, ctx, _, _, vehicle := vehicleRepoFixture(t, "carpool-linked-migration")
	now := time.Now().UTC()
	distance := 10.0
	drive := &models.Drive{VehicleID: vehicle.ID, StartTime: now, EndTime: now, DistanceKm: &distance}
	if err := repo.CreateManualDrive(ctx, drive); err != nil {
		t.Fatal(err)
	}
	linked := &models.CarpoolTrip{VehicleID: vehicle.ID, Title: "linked", Date: now}
	if err := repo.CreateCarpoolTrip(ctx, linked, []models.CarpoolLeg{{DriveID: &drive.ID, DistanceKm: 10}}, nil); err != nil {
		t.Fatal(err)
	}
	manual := &models.CarpoolTrip{VehicleID: vehicle.ID, Title: "manual", Date: now}
	if err := repo.CreateCarpoolTrip(ctx, manual, []models.CarpoolLeg{{DistanceKm: 10}}, nil); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"000071_carpool_linked_costs.down.sql", "000071_carpool_linked_costs.up.sql"} {
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	frozen := func(id string) bool {
		got, err := repo.GetCarpoolTrip(ctx, id, vehicle.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got.CostsFrozenAt != nil
	}
	if !frozen(linked.ID) {
		t.Error("a carpool made from a drive keeps the amounts it was saved with")
	}
	if frozen(manual.ID) {
		t.Error("a carpool entered by distance has nothing to follow")
	}
}
