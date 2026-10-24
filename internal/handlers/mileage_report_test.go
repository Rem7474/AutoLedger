package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestMileageReportTotalsPerTagWithTheUsersScale(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "mileage-report@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "mileage-other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "Car", TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	imp := services.NewCSVImportService(repo, "UTC")
	seed := "start_time,end_time,distance_km,tag\n" +
		"2026-01-10 08:00,2026-01-10 09:00,100,pro\n" +
		"2026-09-15 08:00,2026-09-15 09:00,40,pro\n" +
		"2026-09-16 08:00,2026-09-16 09:00,10,perso\n" +
		"2026-09-17 08:00,2026-09-17 08:30,12,\n"
	if res, err := imp.Execute(ctx, v, []byte(seed), services.CSVImportOptions{Type: services.ImportTypeDrives, DistanceUnit: "km", UserID: u.ID}); err != nil || !res.Committed {
		t.Fatalf("seed: %v %+v", err, res)
	}

	drives, _, err := repo.ListDrives(ctx, v.ID, database.DriveFilter{Tag: "perso"}, 10, 0)
	if err != nil || len(drives) != 1 {
		t.Fatalf("seeded drive: %v %d", err, len(drives))
	}
	if err := repo.UpdateDriveTags(ctx, drives[0].ID, v.ID, []string{"pro", "perso"}); err != nil {
		t.Fatal(err)
	}

	to50 := 50
	for _, m := range []*models.MileageRate{
		{UserID: u.ID, Label: "Scale", Year: 2026, FromKm: 0, ToKm: &to50, RatePerKm: 0.5},
		{UserID: u.ID, Label: "Scale", Year: 2026, FromKm: 50, RatePerKm: 0.3},
		{UserID: other.ID, Label: "Scale", Year: 2026, FromKm: 0, RatePerKm: 9},
	} {
		if err := repo.CreateMileageRate(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	svc := services.NewMileageService(repo, "UTC")
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	report, err := svc.Report(ctx, v, u.ID, services.MileageOptions{From: &from, RateLabel: "Scale"})
	if err != nil {
		t.Fatal(err)
	}
	byTag := map[string]models.MileageTagTotal{}
	for _, row := range report.Tags {
		byTag[row.Tag] = row
	}
	pro := byTag["pro"]
	if pro.Trips != 2 || pro.DistanceKm != 50 {
		t.Fatalf("pro totals: %+v", pro)
	}
	// The 100 km driven in January already used the first slice: the 50 km of September are priced at 0.30.
	if pro.Allowance == nil || pro.Allowance.String() != "15.00" {
		t.Fatalf("pro allowance: %+v", pro.Allowance)
	}
	if byTag["perso"].Trips != 1 || byTag[""].Trips != 1 || byTag[""].DistanceKm != 12 {
		t.Fatalf("other tags: %+v", byTag)
	}
	if report.Tags[0].Tag != "pro" {
		t.Fatalf("tags should be ordered by distance: %+v", report.Tags)
	}

	none, err := svc.Report(ctx, v, u.ID, services.MileageOptions{From: &from, Tag: "perso"})
	if err != nil {
		t.Fatal(err)
	}
	if len(none.Tags) != 1 || none.Tags[0].Allowance != nil {
		t.Fatalf("a tag filter without scale gives one row and no allowance: %+v", none.Tags)
	}

	exp := services.NewExportService(repo).WithMileage(svc)
	out, err := exp.Export(ctx, v, services.ExportOptions{Type: services.ExportMileage, Format: services.ExportCSV, From: &from, UserID: u.ID, RateLabel: "Scale"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out.Body), "pro,2,50,0.00,15.00") {
		t.Fatalf("mileage export: %s", out.Body)
	}

	rates, err := repo.ListMileageRates(ctx, u.ID)
	if err != nil || len(rates) != 2 {
		t.Fatalf("a user lists only their own scales: %v %+v", err, rates)
	}
	if err := repo.DeleteMileageRate(ctx, rates[0].ID, other.ID); err == nil {
		t.Fatal("deleting another user's slice must fail")
	}
}

func TestIntegrationMileageAppliesTheOnlyScaleAndRefusesAnUnknownOne(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "mileage-integration@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "mileage-integration-other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: "Car", TeslaMateAuthType: models.AuthModeNone}
	if err := repo.CreateVehicle(ctx, v); err != nil {
		t.Fatal(err)
	}
	seed := "start_time,end_time,distance_km,tag\n" +
		"2026-01-10 08:00,2026-01-10 09:00,100,pro\n" +
		"2026-09-15 08:00,2026-09-15 09:00,40,pro\n" +
		"2026-09-17 08:00,2026-09-17 08:30,12,\n"
	if res, err := services.NewCSVImportService(repo, "UTC").Execute(ctx, v, []byte(seed), services.CSVImportOptions{Type: services.ImportTypeDrives, DistanceUnit: "km", UserID: u.ID}); err != nil || !res.Committed {
		t.Fatalf("seed: %v %+v", err, res)
	}
	to50 := 50
	for _, m := range []*models.MileageRate{
		{UserID: u.ID, Label: "Scale", Year: 2026, FromKm: 0, ToKm: &to50, RatePerKm: 0.5},
		{UserID: u.ID, Label: "Scale", Year: 2026, FromKm: 50, RatePerKm: 0.3},
		{UserID: u.ID, Label: "Scale", Year: 2025, FromKm: 0, RatePerKm: 0.4},
		{UserID: other.ID, Label: "Theirs", Year: 2026, FromKm: 0, RatePerKm: 9},
	} {
		if err := repo.CreateMileageRate(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	h := NewIntegrationReadHandler(repo, nil).WithMileage(services.NewMileageService(repo, "UTC"), "UTC")

	out, err := h.Mileage(ctx, u.ID, v.ID, "2026-09-01", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Scale == nil || *out.Scale != "Scale" || len(out.Scales) != 1 || len(out.Scales[0].Years) != 2 || out.Scales[0].Years[0] != 2026 {
		t.Fatalf("the only scale is applied and listed with its years, most recent first: %+v", out)
	}
	if out.From != "2026-09-01" || out.Currency != "EUR" || len(out.Tags) != 2 {
		t.Fatalf("report: %+v", out)
	}
	pro := out.Tags[0]
	// 100 km were driven in January: the 40 km of September fall in the second slice, at 0.30.
	if pro.Tag == nil || *pro.Tag != "pro" || pro.Trips != 1 || pro.Distance.Value != 40 || pro.Allowance == nil || pro.Allowance.Amount != "12.00" {
		t.Fatalf("pro row: %+v", pro)
	}
	if untagged := out.Tags[1]; untagged.Tag != nil || untagged.Distance.Value != 12 {
		t.Fatalf("untagged row has a null tag: %+v", untagged)
	}

	if _, err := h.Mileage(ctx, u.ID, v.ID, "", "", "Theirs"); err == nil {
		t.Fatal("another account's scale is unknown")
	}
	if _, err := h.Mileage(ctx, u.ID, v.ID, "yesterday", "", ""); err == nil {
		t.Fatal("a period that is not a date is refused")
	}
	if _, err := h.Mileage(ctx, other.ID, v.ID, "", "", ""); err == nil {
		t.Fatal("another account's vehicle is refused")
	}

	// Two scales: none is guessed, the distances are given and the labels listed.
	if err := repo.CreateMileageRate(ctx, &models.MileageRate{UserID: u.ID, Label: "Second", Year: 2026, FromKm: 0, RatePerKm: 1}); err != nil {
		t.Fatal(err)
	}
	many, err := h.Mileage(ctx, u.ID, v.ID, "2026-09-01", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if many.Scale != nil || len(many.Scales) != 2 || many.Tags[0].Allowance != nil {
		t.Fatalf("without a choice among several scales only distances are given: %+v", many)
	}
}
