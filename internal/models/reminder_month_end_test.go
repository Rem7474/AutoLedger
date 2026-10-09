package models

import (
	"testing"
	"time"
)

func TestReminderMonthlyIntervalsStayInTargetMonth(t *testing.T) {
	for _, tc := range []struct {
		start, want string
		months      int
	}{
		{"2026-01-31", "2026-02-28", 1},
		{"2024-01-31", "2024-02-29", 1},
		{"2024-02-29", "2025-02-28", 12},
		{"2026-03-31", "2026-04-30", 1},
		{"2026-01-15", "2026-02-15", 1},
		{"2026-01-31", "2026-03-31", 2},
	} {
		start, _ := time.Parse("2006-01-02", tc.start)
		reminder := MaintenanceReminder{LastServiceDate: &start, IntervalMonths: &tc.months}
		now, _ := time.Parse("2006-01-02", tc.want)
		reminder.ComputeStatus(0, now.AddDate(0, 0, 1))
		if reminder.DueDate == nil || reminder.DueDate.Format("2006-01-02") != tc.want || reminder.Status != "OVERDUE" {
			t.Fatalf("%s + %d months: %+v, want %s overdue", tc.start, tc.months, reminder, tc.want)
		}
	}
}
