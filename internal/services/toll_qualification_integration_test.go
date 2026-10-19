package services

import (
	"context"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/migrations"
)

func TestIntegrationDrivesNeedingTollQualification(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "qualify@example.com")
	start := time.Date(2026, 9, 18, 18, 0, 0, 0, time.UTC)

	slow := &models.Drive{VehicleID: v.ID, TeslaMateDriveID: intPtr(1), StartTime: start, EndTime: start.Add(30 * time.Minute),
		DistanceKm: kmPtr(20), SpeedAvg: floatPtr(40), SpeedMax: intPtr(90), Tags: []string{}}
	fast := &models.Drive{VehicleID: v.ID, TeslaMateDriveID: intPtr(2), StartTime: start.Add(2 * time.Hour), EndTime: start.Add(3 * time.Hour),
		DistanceKm: kmPtr(120), SpeedAvg: floatPtr(95), SpeedMax: intPtr(130), Tags: []string{}}
	for _, d := range []*models.Drive{slow, fast} {
		if _, err := repo.UpsertTeslaMateDrive(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	needs := func() map[string]bool {
		got, err := repo.DrivesNeedingTollQualification(ctx, v.ID, []string{slow.ID, fast.ID})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if got := needs(); got[slow.ID] || !got[fast.ID] {
		t.Fatalf("only the fast drive looks like a highway drive before any detection, got %v", got)
	}

	found := &models.TollDetection{DriveID: slow.ID, VehicleID: v.ID, Segments: []models.TollSegment{{Type: "open", Entry: "Péage A"}}}
	if err := repo.UpsertTollDetection(ctx, found); err != nil {
		t.Fatal(err)
	}
	if got := needs(); !got[slow.ID] {
		t.Fatal("a drive whose GPS detection found a toll is to qualify, like the list filter says")
	}

	if err := repo.SetDriveTollReviewed(ctx, slow.ID, v.ID, true); err != nil {
		t.Fatal(err)
	}
	if got := needs(); got[slow.ID] || !got[fast.ID] {
		t.Fatalf("a reviewed drive leaves the queue, the other stays: %v", got)
	}

	exp := &models.DriveExpense{VehicleID: v.ID, DriveID: &fast.ID, Type: "TOLL", Amount: 800, Currency: "EUR", Date: fast.StartTime}
	if err := repo.SaveDriveExpense(ctx, exp, nil, ""); err != nil {
		t.Fatal(err)
	}
	if got := needs(); got[fast.ID] {
		t.Fatal("a drive with a toll attached leaves the queue")
	}
	if none, err := repo.DrivesNeedingTollQualification(ctx, v.ID, nil); err != nil || len(none) != 0 {
		t.Fatalf("no drive, no result: %v %v", none, err)
	}
}

func TestIntegrationResetTollDetections(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "reset@example.com")
	start := time.Date(2026, 9, 18, 18, 0, 0, 0, time.UTC)
	d := &models.Drive{VehicleID: v.ID, TeslaMateDriveID: intPtr(1), StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: kmPtr(50), Tags: []string{}}
	if _, err := repo.UpsertTeslaMateDrive(ctx, d); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertTollDetection(ctx, &models.TollDetection{DriveID: d.ID, VehicleID: v.ID, Segments: []models.TollSegment{{Type: "open", Entry: "A"}}}); err != nil {
		t.Fatal(err)
	}
	up, err := migrations.FS.ReadFile("000031_reset_toll_detections.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM toll_detections`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("the stored detections are dropped, %d left", left)
	}
	if got, _ := repo.DrivesNeedingTollQualification(ctx, v.ID, []string{d.ID}); got[d.ID] {
		t.Fatal("without its detection a 50 km drive with no speed data leaves the queue")
	}
}
