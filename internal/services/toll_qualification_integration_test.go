package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/tolldata"
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

func TestIntegrationStopChainNeighboursAndHandPaidTickets(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "chain@example.com")
	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)

	var drives []*models.Drive
	for i := 0; i < 3; i++ {
		from := start.Add(time.Duration(i) * 2 * time.Hour)
		d := &models.Drive{VehicleID: v.ID, TeslaMateDriveID: intPtr(100 + i), StartTime: from, EndTime: from.Add(time.Hour),
			DistanceKm: kmPtr(80), SpeedAvg: floatPtr(95), SpeedMax: intPtr(130), Tags: []string{}}
		if _, err := repo.UpsertTeslaMateDrive(ctx, d); err != nil {
			t.Fatal(err)
		}
		drives = append(drives, d)
	}
	manual := &models.Drive{VehicleID: v.ID, StartTime: start.Add(30 * time.Minute), EndTime: start.Add(45 * time.Minute), DistanceKm: kmPtr(5), Tags: []string{}}
	if err := repo.CreateManualDrive(ctx, manual); err != nil {
		t.Fatal(err)
	}

	next, err := repo.GetNeighbourTeslaMateDrive(ctx, v.ID, drives[0], true)
	if err != nil || next.ID != drives[1].ID {
		t.Fatalf("the next TeslaMate drive follows, a manual drive has no GPS trace: %+v, %v", next, err)
	}
	prev, err := repo.GetNeighbourTeslaMateDrive(ctx, v.ID, drives[2], false)
	if err != nil || prev.ID != drives[1].ID {
		t.Fatalf("the previous drive precedes: %+v, %v", prev, err)
	}
	if _, err := repo.GetNeighbourTeslaMateDrive(ctx, v.ID, drives[0], false); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("nothing before the first drive, got %v", err)
	}
	if _, err := repo.GetNeighbourTeslaMateDrive(ctx, v.ID, drives[2], true); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("nothing after the last drive, got %v", err)
	}

	ticket := tolldata.DriveSegment{Segment: tolldata.Segment{StartMeters: 0, EndMeters: 9000}, Spans: 2}
	chain := &tollChain{
		drives:     drives[:2],
		target:     0,
		targetSegs: []tolldata.DriveSegment{ticket},
		result:     tolldata.ChainResult{PerDrive: [][]tolldata.DriveSegment{{ticket}, {ticket}}},
	}
	s := &TollDetectionService{repo: repo}
	if covered, err := s.sharedTicketPaidByHand(ctx, v, chain); err != nil || covered {
		t.Fatalf("nothing recorded on the other drive yet: %v, %v", covered, err)
	}

	exp := &models.DriveExpense{VehicleID: v.ID, DriveID: &drives[1].ID, Type: "TOLL", Amount: 1200, Currency: "EUR", Date: drives[1].StartTime}
	if err := repo.SaveDriveExpense(ctx, exp, nil, ""); err != nil {
		t.Fatal(err)
	}
	if covered, err := s.sharedTicketPaidByHand(ctx, v, chain); err != nil || !covered {
		t.Fatalf("a toll entered by hand on the other drive covers the journey: %v, %v", covered, err)
	}

	own := &tollChain{drives: drives[:2], target: 0, targetSegs: []tolldata.DriveSegment{{Spans: 1}},
		result: tolldata.ChainResult{PerDrive: [][]tolldata.DriveSegment{{{Spans: 1}}, {}}}}
	if covered, err := s.sharedTicketPaidByHand(ctx, v, own); err != nil || covered {
		t.Fatalf("a ticket that stays on this drive is not covered by another drive's toll: %v, %v", covered, err)
	}
}
