package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

func reminderFixture() []models.MaintenanceReminder {
	km, months := 15000, 12
	lastOdo := 40000.0
	lastDay := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	remKm, remDays := -2500.0, -30
	dueOdo := 55000.0
	dueDay := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	okKm := 9000.0
	observed := 14800
	return []models.MaintenanceReminder{
		{ID: "r1", Title: "Service", Category: "MAINTENANCE", Status: "OVERDUE", IntervalKm: &km, IntervalMonths: &months,
			LastServiceOdometer: &lastOdo, LastServiceDate: &lastDay, DueOdometer: &dueOdo, RemainingKm: &remKm,
			DueDate: &dueDay, RemainingDays: &remDays, ObservedIntervalKm: &observed,
			Maintenance: &models.LinkedMaintenance{Description: "Oil and filters"}},
		{ID: "r2", Title: "Tyres", Category: "TIRES", Status: "DUE_SOON", IntervalKm: &km, RemainingKm: &remKm},
		{ID: "r3", Title: "Inspection", Category: "INSPECTION", Status: "OK", IntervalKm: new(int), RemainingKm: &okKm},
	}
}

func TestIntegrationRemindersReadInKilometresAndCountWhatIsPending(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	out := buildIntegrationReminders("v1", 57500, now, reminderFixture(), "")

	if out.AsOf != "2026-10-19" || out.Odometer != (IntegrationDistance{Value: 57500, Unit: "km"}) || out.Total != 3 || out.Overdue != 1 || out.DueSoon != 1 {
		t.Fatalf("header and counts: %+v", out)
	}
	if len(out.Reminders) != 3 || out.Reminders[0].ID != "r1" {
		t.Fatalf("order of the repository is kept: %+v", out.Reminders)
	}
	first := out.Reminders[0]
	if first.Status != "overdue" || first.Category != "maintenance" ||
		first.IntervalKm == nil || first.IntervalKm.Value != 15000 ||
		first.RemainingKm == nil || first.RemainingKm.Value != -2500 ||
		first.DueDate == nil || *first.DueDate != "2026-09-01" ||
		first.LastServiceDate == nil || *first.LastServiceDate != "2025-09-01" ||
		first.ObservedIntervalKm == nil || first.ObservedIntervalKm.Value != 14800 ||
		first.LastMaintenance == nil || *first.LastMaintenance != "Oil and filters" {
		t.Errorf("an overdue reminder keeps its limits and its last service: %+v", first)
	}
	if out.Reminders[1].IntervalMonths != nil || out.Reminders[1].DueDate != nil || out.Reminders[1].LastMaintenance != nil {
		t.Errorf("what the reminder does not follow is null: %+v", out.Reminders[1])
	}
	if out.Reminders[2].IntervalKm != nil {
		t.Errorf("a zero interval means no interval: %+v", out.Reminders[2].IntervalKm)
	}
}

func TestIntegrationRemindersFilterByStatusWithoutChangingTheCounts(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	out := buildIntegrationReminders("v1", 57500, now, reminderFixture(), "DUE_SOON")

	if len(out.Reminders) != 1 || out.Reminders[0].ID != "r2" {
		t.Fatalf("only the due soon reminders: %+v", out.Reminders)
	}
	if out.Total != 3 || out.Overdue != 1 || out.DueSoon != 1 {
		t.Errorf("the counts ignore the filter: %+v", out)
	}
	none := buildIntegrationReminders("v1", 0, now, nil, "")
	if none.Reminders == nil || len(none.Reminders) != 0 {
		t.Errorf("no reminder is an empty list, not null: %+v", none.Reminders)
	}
}

func TestIntegrationRemindersRejectAnUnknownStatus(t *testing.T) {
	h := &IntegrationReadHandler{}
	if _, err := h.Reminders(t.Context(), "u", "v", "late"); err == nil {
		t.Fatal("an unknown status is refused before anything is read")
	}
}
