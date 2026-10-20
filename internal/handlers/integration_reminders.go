package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// IntegrationReminder is one maintenance reminder of a vehicle with where it stands today.
type IntegrationReminder struct {
	ID       string `json:"id" jsonschema:"identifier of the reminder"`
	Title    string `json:"title" jsonschema:"what the reminder is for, as the owner wrote it"`
	Category string `json:"category" jsonschema:"one of maintenance, tires, inspection, other"`
	Status   string `json:"status" jsonschema:"overdue (the limit is passed), due_soon (inside the warning window) or ok"`
	// The limits are the ones the owner set: a reminder may follow the distance, the calendar, or both.
	IntervalKm     *IntegrationDistance `json:"interval_km" jsonschema:"distance between two services; null when the reminder does not follow the distance"`
	IntervalMonths *int                 `json:"interval_months" jsonschema:"months between two services; null when the reminder does not follow a months interval"`
	// A fixed date replaces the months interval.
	ScheduledDate *string `json:"scheduled_date" jsonschema:"fixed calendar date the reminder is pinned to, YYYY-MM-DD; null when it follows an interval"`
	RepeatYearly  bool    `json:"repeat_yearly" jsonschema:"true when a reminder pinned to a date comes back every year"`

	DueOdometer   *IntegrationDistance `json:"due_odometer" jsonschema:"odometer reading at which the service is due; null without a distance interval"`
	RemainingKm   *IntegrationDistance `json:"remaining_km" jsonschema:"distance left before the service is due, negative once passed; null without a distance interval"`
	DueDate       *string              `json:"due_date" jsonschema:"day the service is due, YYYY-MM-DD; null when the reminder has no date, or nothing is left to do for a one-off date already served"`
	RemainingDays *int                 `json:"remaining_days" jsonschema:"days left before the due date, negative once passed; null without a due date"`

	LastServiceDate     *string              `json:"last_service_date" jsonschema:"day the service was last done, YYYY-MM-DD; null when never recorded, in which case the reminder counts from its creation"`
	LastServiceOdometer *IntegrationDistance `json:"last_service_odometer" jsonschema:"odometer reading when the service was last done; null when never recorded, in which case the reminder counts from zero"`
	// Observed intervals only exist once the service was done at least twice.
	ObservedIntervalKm     *IntegrationDistance `json:"observed_interval_km" jsonschema:"distance really driven between two services, averaged; null below two completions"`
	ObservedIntervalMonths *float64             `json:"observed_interval_months" jsonschema:"months really elapsed between two services, averaged; null below two completions"`
	LastMaintenance        *string              `json:"last_maintenance" jsonschema:"description of the maintenance record the reminder follows from; null when none is linked"`
}

// IntegrationReminders is the maintenance reminders of a vehicle, most urgent first.
type IntegrationReminders struct {
	VehicleID string                `json:"vehicle_id" jsonschema:"the vehicle these reminders belong to"`
	AsOf      string                `json:"as_of" jsonschema:"day the statuses were computed, YYYY-MM-DD"`
	Odometer  IntegrationDistance   `json:"odometer" jsonschema:"odometer reading the distances are measured against"`
	Overdue   int                   `json:"overdue" jsonschema:"number of reminders, filtered or not, whose limit is passed"`
	DueSoon   int                   `json:"due_soon" jsonschema:"number of reminders, filtered or not, inside their warning window"`
	Total     int                   `json:"total" jsonschema:"number of reminders the vehicle has, whatever the filter"`
	Reminders []IntegrationReminder `json:"reminders" jsonschema:"the reminders, overdue first, then due_soon, then ok; within a status the closest in distance first"`
}

var integrationReminderStatuses = map[string]string{"overdue": "OVERDUE", "due_soon": "DUE_SOON", "ok": "OK"}

func reminderDay(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func reminderKm(v *float64) *IntegrationDistance {
	if v == nil {
		return nil
	}
	d := kilometres(*v)
	return &d
}

func reminderIntervalKm(v *int) *IntegrationDistance {
	if v == nil || *v <= 0 {
		return nil
	}
	return reminderKm(func() *float64 { f := float64(*v); return &f }())
}

func reminderIntervalMonths(v *int) *int {
	if v == nil || *v <= 0 {
		return nil
	}
	return v
}

// buildIntegrationReminders shapes the reminders, already sorted by urgency, keeping those of `status` (every one
// when empty). The counters ignore the filter so a caller always sees how much is pending.
func buildIntegrationReminders(vehicleID string, odometer float64, now time.Time, list []models.MaintenanceReminder, status string) IntegrationReminders {
	out := IntegrationReminders{
		VehicleID: vehicleID,
		AsOf:      now.Format("2006-01-02"),
		Odometer:  kilometres(odometer),
		Total:     len(list),
		Reminders: []IntegrationReminder{},
	}
	for _, r := range list {
		switch r.Status {
		case "OVERDUE":
			out.Overdue++
		case "DUE_SOON":
			out.DueSoon++
		}
		if status != "" && r.Status != status {
			continue
		}
		item := IntegrationReminder{
			ID:                     r.ID,
			Title:                  r.Title,
			Category:               strings.ToLower(r.Category),
			Status:                 strings.ToLower(r.Status),
			IntervalKm:             reminderIntervalKm(r.IntervalKm),
			IntervalMonths:         reminderIntervalMonths(r.IntervalMonths),
			ScheduledDate:          reminderDay(r.ScheduledDate),
			RepeatYearly:           r.RepeatYearly,
			DueOdometer:            reminderKm(r.DueOdometer),
			RemainingKm:            reminderKm(r.RemainingKm),
			DueDate:                reminderDay(r.DueDate),
			RemainingDays:          r.RemainingDays,
			LastServiceDate:        reminderDay(r.LastServiceDate),
			LastServiceOdometer:    reminderKm(r.LastServiceOdometer),
			ObservedIntervalKm:     reminderIntervalKm(r.ObservedIntervalKm),
			ObservedIntervalMonths: r.ObservedIntervalMonths,
		}
		if r.Maintenance != nil {
			description := r.Maintenance.Description
			item.LastMaintenance = &description
		}
		out.Reminders = append(out.Reminders, item)
	}
	return out
}

// Reminders lists the maintenance reminders of a vehicle. status is overdue, due_soon, ok, or empty for all.
func (h *IntegrationReadHandler) Reminders(ctx context.Context, userID, vehicleID, status string) (*IntegrationReminders, error) {
	want := ""
	if status != "" {
		var ok bool
		if want, ok = integrationReminderStatuses[strings.ToLower(status)]; !ok {
			return nil, apierror.New("request.invalid_status", "Status must be overdue, due_soon or ok")
		}
	}
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	list, err := h.repo.ListMaintenanceReminders(ctx, vehicleID, vehicle.CurrentOdometer)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationReminders(vehicleID, vehicle.CurrentOdometer, time.Now(), list, want)
	return &out, nil
}

// GetReminders serves Reminders.
func (h *IntegrationReadHandler) GetReminders(w http.ResponseWriter, r *http.Request) {
	out, err := h.Reminders(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), r.URL.Query().Get("status"))
	if err != nil {
		if apiErr, ok := apierror.As(err); ok {
			writeAPIError(w, http.StatusBadRequest, apiErr)
			return
		}
		writeIntegrationError(w, r, err, "Failed to list the reminders")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
