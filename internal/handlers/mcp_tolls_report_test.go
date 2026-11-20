package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestCarpoolWhereNarrowsToAPeriodOrOneCarpoolAndNamesItsDrives(t *testing.T) {
	a := newCarpoolAPI(t, "carpoolperiod")
	ctx := context.Background()
	h := writeHandler(a)
	trip := recordedCarpool(t, a, h) // 2026-09-05, two legs

	if len(trip.DriveIDs) != 2 {
		t.Fatalf("a carpool names the trips that are its legs: %+v", trip.DriveIDs)
	}
	for _, c := range []struct {
		name, from, to, id  string
		trips, summaryTrips int
	}{
		{"whole history", "", "", "", 1, 1},
		{"period holding it", "2026-09-01", "2026-09-05", "", 1, 1},
		{"period before it", "2026-08-01", "2026-09-04", "", 0, 0},
		{"period after it", "2026-09-06", "", "", 0, 0},
		{"its id", "", "", trip.ID, 1, 1},
		{"another id", "", "", "00000000-0000-0000-0000-000000000000", 0, 1},
	} {
		out, err := h.CarpoolWhere(ctx, a.userID, a.vid, 0, c.from, c.to, c.id)
		if err != nil || len(out.Trips) != c.trips || out.Summary.Trips != c.summaryTrips {
			t.Errorf("%s: %d trips, summary %d, %v", c.name, len(out.Trips), out.Summary.Trips, err)
		}
	}
	out, _ := h.CarpoolWhere(ctx, a.userID, a.vid, 0, "", "", trip.ID)
	if got := out.Trips[0].DriveIDs; len(got) != 2 || got[0] != trip.DriveIDs[0] || got[1] != trip.DriveIDs[1] {
		t.Errorf("the drives come in leg order: %v", got)
	}
	if _, err := h.CarpoolWhere(ctx, a.userID, a.vid, 0, "09/05/2026", "", ""); err == nil {
		t.Error("a date that is not YYYY-MM-DD is refused")
	}
}

func TestTollsReportListsTheTollsOfAPeriodWithTheirTripsAndCarpools(t *testing.T) {
	a := newCarpoolAPI(t, "tollsreport")
	ctx := context.Background()
	h := writeHandler(a)
	trip := recordedCarpool(t, a, h)
	day := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	save := func(driveID string, kind string, amount int64, date time.Time, source string) {
		t.Helper()
		e := &models.DriveExpense{VehicleID: a.vid, Type: kind, Amount: money.Cents(amount), Currency: "EUR", Date: date, Source: source}
		if driveID != "" {
			e.DriveID = &driveID
		}
		if err := a.repo.SaveDriveExpense(ctx, e, nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	other := manualDrive(t, a, time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC), "Paris, France", "Lille, France", 220)
	save(trip.DriveIDs[0], "TOLL", 1460, day, models.ExpenseSourceAutoToll)
	save(trip.DriveIDs[1], "TOLL", 900, day.Add(2*time.Hour), models.ExpenseSourceManual)
	save(trip.DriveIDs[1], "PARKING", 350, day.Add(2*time.Hour), models.ExpenseSourceManual)
	save(other.ID, "TOLL", 2000, time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC), models.ExpenseSourceManual)

	all, err := h.TollsReport(ctx, a.userID, a.vid, "", "", 0)
	if err != nil || all.Count != 3 || all.Total.Amount != "43.60" || all.OtherCurrency != 0 || all.From != nil || all.To != nil {
		t.Fatalf("every toll, parking left out: %+v %v", all, err)
	}
	if all.Tolls[0].Amount.Amount != "20.00" || len(all.Tolls[0].CarpoolIDs) != 0 || all.Tolls[0].DriveIDs[0] != other.ID {
		t.Errorf("the most recent first, a toll of no carpool lists none: %+v", all.Tolls[0])
	}

	sept, err := h.TollsReport(ctx, a.userID, a.vid, "2026-09-01", "2026-09-05", 0)
	if err != nil || sept.Count != 2 || sept.Total.Amount != "23.60" || sept.From == nil || *sept.From != "2026-09-01" || *sept.To != "2026-09-05" {
		t.Fatalf("a period keeps the tolls dated within it, its last day included: %+v %v", sept, err)
	}
	if sept.Tolls[0].Source != "entered" || sept.Tolls[1].Source != "estimated" {
		t.Errorf("provenance: %q %q", sept.Tolls[0].Source, sept.Tolls[1].Source)
	}
	for _, row := range sept.Tolls {
		if len(row.CarpoolIDs) != 1 || row.CarpoolIDs[0] != trip.ID {
			t.Errorf("a toll of a leg names its carpool: %+v", row)
		}
	}

	cut, err := h.TollsReport(ctx, a.userID, a.vid, "", "", 1)
	if err != nil || !cut.Truncated || len(cut.Tolls) != 1 || cut.Count != 3 || cut.Total.Amount != "43.60" {
		t.Errorf("the limit cuts the list, never the total: %+v %v", cut, err)
	}
	if _, err := h.TollsReport(ctx, a.userID, a.vid, "", "yesterday", 0); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("a bad date is refused: %v", err)
	}
}
