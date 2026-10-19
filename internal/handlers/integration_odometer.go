package handlers

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationOdometerEstimate is the odometer the records give for one day.
type IntegrationOdometerEstimate struct {
	Date     string              `json:"date" jsonschema:"the day asked, YYYY-MM-DD"`
	Odometer IntegrationDistance `json:"odometer" jsonschema:"the odometer reading on that day"`
	Source   string              `json:"source" jsonschema:"exact (a reading of that day), interpolated (spread in time between the readings on either side) or bounded (the day is before the first or after the last reading, so the nearest reading is kept)"`
}

// IntegrationOdometer is what the odometer readings say about a vehicle.
type IntegrationOdometer struct {
	VehicleID    string                       `json:"vehicle_id" jsonschema:"the vehicle these readings belong to"`
	Current      IntegrationDistance          `json:"current" jsonschema:"the odometer as last recorded"`
	Readings     int                          `json:"readings" jsonschema:"dated odometer readings the records hold (drives, checkpoints, fill-ups, ownership start)"`
	FirstReading *string                      `json:"first_reading" jsonschema:"day of the oldest reading, YYYY-MM-DD; null when there is none"`
	LastReading  *string                      `json:"last_reading" jsonschema:"day of the most recent reading, YYYY-MM-DD; null when there is none"`
	Estimate     *IntegrationOdometerEstimate `json:"estimate" jsonschema:"the odometer on the day asked for; null when no day was asked or the records hold no usable reading"`
}

func buildIntegrationOdometer(vehicleID string, current float64, anchors []services.OdometerAnchor, now time.Time, at *time.Time) IntegrationOdometer {
	out := IntegrationOdometer{VehicleID: vehicleID, Current: kilometres(roundTo(current, 0))}
	dated := []time.Time{}
	for _, a := range anchors {
		// The current odometer is dated now: it is not a reading of its own
		if a.Km > 0 && a.Date.Before(now.Add(-time.Minute)) {
			dated = append(dated, a.Date)
		}
	}
	sort.Slice(dated, func(i, j int) bool { return dated[i].Before(dated[j]) })
	out.Readings = len(dated)
	if len(dated) > 0 {
		first, last := dated[0].UTC().Format(time.DateOnly), dated[len(dated)-1].UTC().Format(time.DateOnly)
		out.FirstReading, out.LastReading = &first, &last
	}
	if at != nil {
		if km, source, ok := services.EstimateOdometerAt(anchors, *at); ok {
			out.Estimate = &IntegrationOdometerEstimate{Date: at.UTC().Format(time.DateOnly), Odometer: kilometres(math.Round(km)), Source: source}
		}
	}
	return out
}

// Odometer gives the current odometer, the readings behind it and, for a day, the odometer the records give.
func (h *IntegrationReadHandler) Odometer(ctx context.Context, userID, vehicleID, day string) (*IntegrationOdometer, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	var at *time.Time
	if day = strings.TrimSpace(day); day != "" {
		parsed, err := parseDate(day, time.UTC)
		if err != nil {
			return nil, err
		}
		// A day runs to its end: the readings of that day count
		if len(day) == len(time.DateOnly) {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Second)
		}
		at = &parsed
	}
	stored, err := h.repo.ListOdometerAnchors(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	anchors := make([]services.OdometerAnchor, len(stored))
	for i, a := range stored {
		anchors[i] = services.OdometerAnchor{Date: a.Date, Km: a.Km, Origin: a.Origin}
	}
	out := buildIntegrationOdometer(vehicle.ID, vehicle.CurrentOdometer, anchors, time.Now(), at)
	return &out, nil
}

// GetOdometer serves Odometer.
func (h *IntegrationReadHandler) GetOdometer(w http.ResponseWriter, r *http.Request) {
	out, err := h.Odometer(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), r.URL.Query().Get("date"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to read the odometer")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
