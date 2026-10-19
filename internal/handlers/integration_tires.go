package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// WithTires gives the tires view the wear statistics.
func (h *IntegrationReadHandler) WithTires(s *services.TireWearService) *IntegrationReadHandler {
	h.tireWear = s
	return h
}

// IntegrationTire is one tire or set of the vehicle and how worn it is.
type IntegrationTire struct {
	Brand             string               `json:"brand" jsonschema:"brand as the owner typed it, user text; may be empty"`
	Model             string               `json:"model" jsonschema:"model as the owner typed it, user text; may be empty"`
	Dimension         string               `json:"dimension" jsonschema:"size, such as 225/45 R18, user text"`
	Season            string               `json:"season" jsonschema:"SUMMER, WINTER or ALL_SEASON"`
	Position          string               `json:"position" jsonschema:"FL, FR, RL or RR when mounted on that wheel, STORAGE when kept off the car, DISPOSED when scrapped"`
	PurchaseDate      string               `json:"purchase_date" jsonschema:"day the tire was bought, YYYY-MM-DD"`
	PurchasePrice     IntegrationAmount    `json:"purchase_price" jsonschema:"what it cost"`
	Distance          IntegrationDistance  `json:"distance" jsonschema:"distance driven on this tire, mount sessions and the distance it had before it was recorded included"`
	LifeUsedPct       float64              `json:"life_used_pct" jsonschema:"share of the expected lifespan already driven, in percent; can exceed 100"`
	TreadDepthMm      float64              `json:"tread_depth_mm" jsonschema:"tread depth: the latest measurement, or an estimate when none was recorded (see depth_measured)"`
	DepthMeasured     bool                 `json:"depth_measured" jsonschema:"true when tread_depth_mm comes from a measurement the owner logged, false when it is estimated"`
	LegalMinMm        float64              `json:"legal_min_mm" jsonschema:"legal minimum tread depth"`
	WearPct           float64              `json:"wear_pct" jsonschema:"share of the usable tread (from new down to the legal minimum) already worn, in percent"`
	Condition         string               `json:"condition" jsonschema:"GOOD, WARNING or CRITICAL"`
	RemainingKm       *IntegrationDistance `json:"remaining_distance" jsonschema:"distance left before the legal minimum, from the measured wear when there is any; null once the tire is disposed of"`
	WearBasis         string               `json:"wear_basis" jsonschema:"where the wear rate behind remaining_distance comes from: measured (depth logs) or default (average wear)"`
	ReplacementDate   *string              `json:"replacement_date" jsonschema:"day the tire is expected to reach the legal minimum, YYYY-MM-DD, from the distance driven per month while it is on the car; null when it cannot be predicted"`
	CostPerKm         *IntegrationAmount   `json:"cost_per_km" jsonschema:"purchase price over the distance driven; null while no distance is driven"`
	DepthMeasurements int                  `json:"depth_measurements" jsonschema:"how many tread depth measurements were logged"`
}

// IntegrationTires lists the tires of a vehicle.
type IntegrationTires struct {
	VehicleID string            `json:"vehicle_id" jsonschema:"the vehicle these tires belong to"`
	Tires     []IntegrationTire `json:"tires" jsonschema:"the tires and sets of the vehicle that are not archived: mounted ones first, then the ones in storage"`
}

func buildIntegrationTires(vehicleID, currency string, stats []services.TireWearStats) IntegrationTires {
	out := IntegrationTires{VehicleID: vehicleID, Tires: []IntegrationTire{}}
	for _, s := range stats {
		t := s.Tire
		item := IntegrationTire{
			Brand: t.Brand, Model: t.Model, Dimension: t.Dimension, Season: string(t.Season), Position: string(t.CurrentPosition),
			PurchaseDate: t.PurchaseDate.UTC().Format(time.DateOnly), PurchasePrice: amountOf(t.PurchasePrice, currency),
			Distance: kilometres(s.TotalDistanceKm), LifeUsedPct: roundTo(s.LifeProgressPct, 1), TreadDepthMm: roundTo(s.CurrentDepthMm, 1),
			DepthMeasured: s.LogsCount > 0, LegalMinMm: s.MinLegalDepthMm, WearPct: roundTo(s.WearPercentage, 1), Condition: s.Condition,
			WearBasis: s.WearRateSource, DepthMeasurements: s.LogsCount,
		}
		if t.CurrentPosition != models.TirePosDisposed {
			remaining := s.EstimatedRemainingKm
			if s.DrivesCount > 0 {
				remaining = s.DynamicRemainingKm
			}
			km := kilometres(roundTo(remaining, 0))
			item.RemainingKm = &km
		}
		if f := s.ReplacementForecast; f != nil && f.ReplacementDate != "" {
			item.ReplacementDate = &f.ReplacementDate
		}
		if s.TotalDistanceKm > 0 && s.CostPerKm > 0 {
			item.CostPerKm = decimalAmount(&s.CostPerKm, currency, 4)
		}
		out.Tires = append(out.Tires, item)
	}
	return out
}

// Tires gives the wear of every tire of a vehicle and when each is expected to reach its legal minimum.
func (h *IntegrationReadHandler) Tires(ctx context.Context, userID, vehicleID string) (*IntegrationTires, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if h.tireWear == nil {
		return nil, apierror.New("internal", "Tires are unavailable")
	}
	tires, err := h.repo.ListTires(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	stats := []services.TireWearStats{}
	for i := range tires {
		s, err := h.tireWear.CalculateTireWear(ctx, &tires[i], vehicle.CurrentOdometer)
		if err == nil && s != nil {
			stats = append(stats, *s)
		}
	}
	attachTireForecasts(ctx, h.repo, vehicle.ID, stats)
	out := buildIntegrationTires(vehicle.ID, vehicle.Currency, stats)
	return &out, nil
}

// GetTires serves Tires.
func (h *IntegrationReadHandler) GetTires(w http.ResponseWriter, r *http.Request) {
	out, err := h.Tires(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the tire wear")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
