package handlers

import (
	"testing"
	"time"
)

func TestCalendarHandlerTimezoneDefaultsAndConfiguration(t *testing.T) {
	handler := NewFuelHandler(nil)
	if handler.dateLocation() != time.UTC {
		t.Fatal("unconfigured handlers must default to UTC")
	}
	handler.SetTimezone("America/Argentina/Buenos_Aires")
	log, err := buildFuelLog("vehicle", &SaveFuelLogRequest{Date: "2026-10-01", Amount: cents(100)}, handler.dateLocation())
	if err != nil || log.Date.UTC().Hour() != 3 {
		t.Fatalf("configured manual calendar: %+v %v", log, err)
	}
	handler.SetTimezone("Nowhere/Land")
	if handler.dateLocation() != time.UTC {
		t.Fatal("invalid timezone must fall back to UTC")
	}
	if _, err := parseDate("invalid", handler.dateLocation()); err == nil {
		t.Fatal("invalid date accepted")
	}
}

func TestManualDatesFollowReportingCalendar(t *testing.T) {
	for _, zone := range []string{"America/Argentina/Buenos_Aires", "Europe/Paris", "UTC"} {
		loc, _ := time.LoadLocation(zone)
		fuel, err := buildFuelLog("vehicle", &SaveFuelLogRequest{Date: "2026-10-01", Amount: cents(100), Liters: f64(50)}, loc)
		if err != nil || fuel.Date.In(loc).Format("2006-01-02 15:04") != "2026-10-01 00:00" {
			t.Fatalf("%s fuel date: %+v %v", zone, fuel, err)
		}
		stamp := "2026-10-01T01:00:00Z"
		got, err := parseDate(stamp, loc)
		want, _ := time.Parse(time.RFC3339, stamp)
		if err != nil || !got.Equal(want) {
			t.Fatalf("explicit instant must be preserved: %v %v", got, err)
		}
	}
}

func TestExportDatesCoverLocalDaysIncludingDST(t *testing.T) {
	for _, tc := range []struct {
		zone, day string
		hours     int
	}{
		{"America/Argentina/Buenos_Aires", "2026-10-01", 24},
		{"Europe/Paris", "2026-03-29", 23},
		{"Europe/Paris", "2026-10-25", 25},
	} {
		loc, _ := time.LoadLocation(tc.zone)
		from, err := parseExportDate(tc.day, false, loc)
		if err != nil {
			t.Fatal(err)
		}
		to, err := parseExportDate(tc.day, true, loc)
		if err != nil {
			t.Fatal(err)
		}
		late, _ := time.ParseInLocation("2006-01-02 15:04", tc.day+" 23:59", loc)
		if from.Format("2006-01-02 15:04") != tc.day+" 00:00" || late.After(*to) || to.Add(time.Nanosecond).Sub(*from) != time.Duration(tc.hours)*time.Hour {
			t.Fatalf("local day %s in %s: %v..%v", tc.day, tc.zone, from, to)
		}
	}
	if value, err := parseExportDate("", false); value != nil || err != nil {
		t.Fatal("empty date must remain optional")
	}
	if _, err := parseExportDate("2026-02-30", false); err == nil {
		t.Fatal("invalid calendar date accepted")
	}
}
