package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/services"
)

// WithMileage gives the mileage view the service that prices the distance with a scale, and the timezone that
// decides which day a drive belongs to.
func (h *IntegrationReadHandler) WithMileage(s *services.MileageService, zone string) *IntegrationReadHandler {
	h.mileage = s
	h.SetTimezone(zone)
	return h
}

// IntegrationMileageScale is a mileage allowance scale the owner typed, with the years it has rates for.
type IntegrationMileageScale struct {
	Label string `json:"label" jsonschema:"name the owner gave the scale, user text; pass it as rates"`
	Years []int  `json:"years" jsonschema:"the years the scale has slices for, most recent first"`
}

// IntegrationMileageTag is the distance of the drives carrying one tag and the allowance it earns.
type IntegrationMileageTag struct {
	Tag       *string             `json:"tag" jsonschema:"the tag as the owner typed it, user text; null for the drives that carry no tag"`
	Trips     int                 `json:"trips" jsonschema:"number of drives with this tag in the period"`
	Distance  IntegrationDistance `json:"distance" jsonschema:"distance of those drives"`
	Tolls     IntegrationAmount   `json:"tolls" jsonschema:"tolls and parking attributed to those drives"`
	Allowance *IntegrationAmount  `json:"allowance" jsonschema:"the scale applied to this tag's own cumulative distance of each year, so the slices restart on 1 January; null when no scale was applied"`
}

// IntegrationMileageReport is the distance driven per tag over a period, with the allowance of a scale the owner typed.
type IntegrationMileageReport struct {
	VehicleID string                    `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string                    `json:"currency" jsonschema:"currency of every amount"`
	From      string                    `json:"from" jsonschema:"first day counted, YYYY-MM-DD in the account's timezone; empty when the period has no start"`
	To        string                    `json:"to" jsonschema:"last day counted, YYYY-MM-DD in the account's timezone; empty when the period has no end"`
	Scale     *string                   `json:"rate_label" jsonschema:"the mileage allowance scale applied, user text; null when none was applied"`
	Scales    []IntegrationMileageScale `json:"available_scales" jsonschema:"every scale the owner typed (their own rates, not an official table), so the one to ask for can be named"`
	Tags      []IntegrationMileageTag   `json:"tags" jsonschema:"one row per tag plus one for the untagged drives, longest distance first; a drive with several tags counts in each, so rows are not added up"`
}

func mileageScales(h *IntegrationReadHandler, ctx context.Context, userID string) ([]IntegrationMileageScale, error) {
	rates, err := h.repo.ListMileageRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	years := map[string]map[int]bool{}
	for _, r := range rates {
		if years[r.Label] == nil {
			years[r.Label] = map[int]bool{}
		}
		years[r.Label][r.Year] = true
	}
	out := []IntegrationMileageScale{}
	for label, ys := range years {
		scale := IntegrationMileageScale{Label: label, Years: []int{}}
		for y := range ys {
			scale.Years = append(scale.Years, y)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(scale.Years)))
		out = append(out, scale)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out, nil
}

// Mileage totals a vehicle's drives per tag over a period (From and To are days, YYYY-MM-DD, both optional) and
// prices them with the scale named by rates. Without rates, the only scale the owner typed is applied.
func (h *IntegrationReadHandler) Mileage(ctx context.Context, userID, vehicleID, from, to, rates string) (*IntegrationMileageReport, error) {
	if h.mileage == nil {
		return nil, apierror.New("internal", "The mileage report is unavailable")
	}
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	start, err := parseExportDate(from, false, h.dateLocation())
	if err != nil {
		return nil, err
	}
	end, err := parseExportDate(to, true, h.dateLocation())
	if err != nil {
		return nil, err
	}
	scales, err := mileageScales(h, ctx, userID)
	if err != nil {
		return nil, err
	}
	rates = strings.TrimSpace(rates)
	switch {
	case rates == "" && len(scales) == 1:
		rates = scales[0].Label
	case rates != "" && !hasMileageScale(scales, rates):
		return nil, apierror.New("request.invalid_value", "Unknown mileage scale: pass rates as one of the labels in available_scales, or leave it out for the distances alone")
	}
	report, err := h.mileage.Report(ctx, vehicle, userID, services.MileageOptions{From: start, To: end, RateLabel: rates})
	if err != nil {
		return nil, err
	}
	out := &IntegrationMileageReport{
		VehicleID: vehicle.ID, Currency: vehicle.Currency, From: report.From, To: report.To,
		Scales: scales, Tags: []IntegrationMileageTag{},
	}
	if rates != "" {
		out.Scale = &rates
	}
	for _, t := range report.Tags {
		row := IntegrationMileageTag{Trips: t.Trips, Distance: kilometres(t.DistanceKm), Tolls: amountOf(t.Tolls, vehicle.Currency)}
		if t.Tag != "" {
			tag := t.Tag
			row.Tag = &tag
		}
		if t.Allowance != nil {
			a := amountOf(*t.Allowance, vehicle.Currency)
			row.Allowance = &a
		}
		out.Tags = append(out.Tags, row)
	}
	return out, nil
}

func hasMileageScale(scales []IntegrationMileageScale, label string) bool {
	for _, s := range scales {
		if s.Label == label {
			return true
		}
	}
	return false
}

// GetMileage serves Mileage.
func (h *IntegrationReadHandler) GetMileage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	out, err := h.Mileage(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), query.Get("from"), query.Get("to"), query.Get("rates"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to build the mileage report")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
