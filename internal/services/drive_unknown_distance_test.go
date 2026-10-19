package services

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

// mustDriveWithoutDistance stores a drive an integration reported with a start and an end but no distance.
func mustDriveWithoutDistance(t *testing.T, repo *database.Repository, vehicleID string, tmID int, start time.Time) *models.Drive {
	t.Helper()
	d := &models.Drive{VehicleID: vehicleID, Origin: "WEBHOOK", TeslaMateDriveID: &tmID, StartTime: start, EndTime: start.Add(time.Hour), Tags: []string{}}
	if _, err := repo.UpsertTeslaMateDrive(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestIntegrationDriveWithoutDistanceIsStoredUnknownAndLeavesTheSumsAlone(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "unknown-distance@example.com")
	base := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)

	known := mustDrive(t, repo, v.ID, 1, base, 10000, 100)
	unknown := mustDriveWithoutDistance(t, repo, v.ID, 2, base.Add(3*time.Hour))
	alone := mustDriveWithoutDistance(t, repo, v.ID, 3, base.Add(48*time.Hour))

	// Stored as NULL and read back as unknown, not as a drive of zero kilometres.
	var stored *float64
	if err := db.Pool.QueryRow(ctx, `SELECT distance_km FROM drives WHERE id = $1`, unknown.ID).Scan(&stored); err != nil || stored != nil {
		t.Fatalf("stored distance: %v, %v", stored, err)
	}
	got, err := repo.GetDriveByID(ctx, unknown.ID, v.ID)
	if err != nil || got.DistanceKm != nil || got.Distance() != 0 {
		t.Fatalf("read back: %+v, %v", got.DistanceKm, err)
	}

	// A toll over a group holding a known and an unknown drive goes to the known one: the unknown drive has no weight,
	// and its missing distance must not make the whole allocation unknown.
	toll := &models.DriveExpense{VehicleID: v.ID, Type: "TOLL", Amount: 4000, Currency: "EUR", Date: base}
	if err := repo.SaveDriveExpense(ctx, toll, []string{known.ID, unknown.ID}, "Out and back"); err != nil {
		t.Fatal(err)
	}
	alloc, err := repo.GetTollExpensesForDrives(ctx, v.ID, []string{known.ID, unknown.ID})
	if err != nil {
		t.Fatal(err)
	}
	if alloc[known.ID] != 4000 || alloc[unknown.ID] != 0 {
		t.Fatalf("allocation: %v", alloc)
	}
	// A group where no drive has a distance shares the toll equally.
	other := mustDriveWithoutDistance(t, repo, v.ID, 4, base.Add(72*time.Hour))
	toll2 := &models.DriveExpense{VehicleID: v.ID, Type: "TOLL", Amount: 1000, Currency: "EUR", Date: base.Add(48 * time.Hour)}
	if err := repo.SaveDriveExpense(ctx, toll2, []string{alone.ID, other.ID}, "No distance"); err != nil {
		t.Fatal(err)
	}
	alloc, err = repo.GetTollExpensesForDrives(ctx, v.ID, []string{alone.ID, other.ID})
	if err != nil || alloc[alone.ID] != 500 || alloc[other.ID] != 500 {
		t.Fatalf("equal split: %v, %v", alloc, err)
	}
	groups, err := repo.ListTripGroups(ctx, v.ID)
	if err != nil || len(groups) != 2 {
		t.Fatalf("trip groups: %d, %v", len(groups), err)
	}

	// The totals count only the known distance and still read when no drive has one.
	tco := NewTCOService(db.Pool, "UTC")
	if _, err := tco.ComputeVehicleTCO(ctx, v.ID); err != nil {
		t.Fatalf("tco: %v", err)
	}
	if _, _, err := repo.ListDrives(ctx, v.ID, database.DriveFilter{}, 50, 0); err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, err := repo.CountUnqualifiedDrives(ctx, v.ID); err != nil {
		t.Fatalf("unqualified: %v", err)
	}
	if _, err := repo.ListTripCandidateDrives(ctx, v.ID, base.AddDate(0, 0, -1)); err != nil {
		t.Fatalf("trip candidates: %v", err)
	}
}
