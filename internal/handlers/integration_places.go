package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// Caps of the charging places view: a response never grows with the history.
const (
	integrationPlacesDefaultMonths = 12
	integrationPlacesMaxMonths     = 240
	integrationPlaceAddresses      = 10
	integrationUnplacedLocations   = 20
)

// IntegrationChargingPlace is where a vehicle charged, with what the charging there cost.
type IntegrationChargingPlace struct {
	Name         *string            `json:"name" jsonschema:"the place's name as the owner gave it, user text; null for the charges linked to no place"`
	Category     string             `json:"category" jsonschema:"HOME, WORK, PUBLIC, OTHER, OUTSIDE (a charge away from the home place that is linked to no place) or UNPLACED (no location reported, or no place defined)"`
	Addresses    []string           `json:"addresses" jsonschema:"the locations charges reported that resolve to this place, user text; at most 10, the most used first; empty when none is known"`
	Sessions     int                `json:"sessions" jsonschema:"charging sessions that added energy, in the period"`
	EnergyKwh    float64            `json:"energy_kwh" jsonschema:"energy those sessions added, in kWh"`
	Cost         IntegrationAmount  `json:"cost" jsonschema:"what those sessions cost, free ones counted as zero; sessions without a cost are left out"`
	PricePerKwh  *IntegrationAmount `json:"price_per_kwh" jsonschema:"cost over the energy of the sessions whose cost is known; null while none is known"`
	FreeSessions int                `json:"free_sessions" jsonschema:"sessions recorded as free"`
	Unpriced     int                `json:"unpriced_sessions" jsonschema:"sessions with no recorded cost, so neither in the cost nor in the price"`
	UnpricedKwh  float64            `json:"unpriced_energy_kwh" jsonschema:"energy of those sessions, in kWh"`
}

// IntegrationHomeSavings compares what charging away from home cost with the same energy at the home price.
type IntegrationHomeSavings struct {
	HomePricePerKwh *IntegrationAmount `json:"home_price_per_kwh" jsonschema:"price per kWh at the home place; null when unknown"`
	AwaySessions    int                `json:"away_sessions" jsonschema:"sessions away from home"`
	AwayCost        IntegrationAmount  `json:"away_cost" jsonschema:"what charging away from home cost"`
	AtHomePrice     IntegrationAmount  `json:"at_home_price" jsonschema:"what the same energy would have cost at the home price"`
	Savings         IntegrationAmount  `json:"savings" jsonschema:"away cost minus the cost at the home price: positive when charging at home would have been cheaper, negative when charging away was"`
}

// IntegrationUnplacedLocation is a location charges reported that is not linked to any place.
type IntegrationUnplacedLocation struct {
	Text      string  `json:"text" jsonschema:"the location as the charges reported it, user text"`
	Sessions  int     `json:"sessions" jsonschema:"charging sessions that reported it"`
	EnergyKwh float64 `json:"energy_kwh" jsonschema:"energy those sessions added, in kWh"`
}

// IntegrationChargingPlaces is the charging of a vehicle by place.
type IntegrationChargingPlaces struct {
	VehicleID string                        `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string                        `json:"currency" jsonschema:"currency of every amount"`
	Months    int                           `json:"months" jsonschema:"the current month and the months before it that the figures cover; 0 means the whole history"`
	Places    []IntegrationChargingPlace    `json:"places" jsonschema:"one entry per place that vehicle charged at in the period, the one with most energy first"`
	Home      *IntegrationHomeSavings       `json:"home_savings" jsonschema:"what charging away from home cost compared with the home price; null when nothing can be compared"`
	HomeWhy   *string                       `json:"home_savings_unavailable" jsonschema:"why home_savings is null: no_home (no priced charge at a home place) or no_away (no priced charge away from home); null when it is computed"`
	Unplaced  []IntegrationUnplacedLocation `json:"unplaced_locations" jsonschema:"the 20 most used locations over the whole history that are linked to no place: the owner could name them"`
}

func categoryOfGroup(g services.PlaceGroup) string {
	if g.Category == "" {
		return "UNPLACED"
	}
	return g.Category
}

// buildIntegrationPlaces folds the place statistics and the known locations into the view.
func buildIntegrationPlaces(vehicleID, currency string, months int, stats *services.ChargingPlaceStats, locations []models.ChargingLocation) IntegrationChargingPlaces {
	out := IntegrationChargingPlaces{VehicleID: vehicleID, Currency: currency, Months: months, Places: []IntegrationChargingPlace{}, Unplaced: []IntegrationUnplacedLocation{}}

	addresses := map[string][]string{}
	for _, l := range locations {
		text := publicAddress(&l.Text)
		if text == nil {
			continue
		}
		if l.PlaceID == nil {
			if len(out.Unplaced) < integrationUnplacedLocations {
				out.Unplaced = append(out.Unplaced, IntegrationUnplacedLocation{Text: *text, Sessions: l.Sessions, EnergyKwh: roundTo(l.KwhAdded, 1)})
			}
			continue
		}
		if len(addresses[*l.PlaceID]) < integrationPlaceAddresses {
			addresses[*l.PlaceID] = append(addresses[*l.PlaceID], *text)
		}
	}

	for _, g := range stats.Groups {
		item := IntegrationChargingPlace{
			Category: categoryOfGroup(g), Addresses: []string{}, Sessions: g.Sessions, EnergyKwh: g.KwhAdded, Cost: amountOf(g.Cost, currency),
			PricePerKwh: decimalAmount(g.PricePerKwh, currency, 4), FreeSessions: g.FreeSessions, Unpriced: g.UnknownSessions, UnpricedKwh: g.UnknownKwh,
		}
		if g.Name != "" {
			name := g.Name
			item.Name = &name
		}
		if found := addresses[g.Key]; g.Key != "" && found != nil {
			item.Addresses = found
		}
		out.Places = append(out.Places, item)
	}

	h := stats.HomeSavings
	if h.Reason != "" {
		out.HomeWhy = &h.Reason
	} else {
		out.Home = &IntegrationHomeSavings{
			HomePricePerKwh: decimalAmount(h.HomePricePerKwh, currency, 4), AwaySessions: h.AwaySessions,
			AwayCost: amountOf(h.AwayCost, currency), AtHomePrice: amountOf(h.HypotheticalCost, currency), Savings: amountOf(h.Savings, currency),
		}
	}
	return out
}

// ChargingPlaces groups the charging of a vehicle by place over the current and the previous months (0 = the whole
// history).
func (h *IntegrationReadHandler) ChargingPlaces(ctx context.Context, userID, vehicleID string, months int) (*IntegrationChargingPlaces, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if months < 0 || months > integrationPlacesMaxMonths {
		return nil, apierror.New("request.invalid_months", "months must be between 0 and 240")
	}
	period := services.PlacePeriod{Months: months}
	stats := services.EmptyPlaceStats(period)
	var locations []models.ChargingLocation
	if vehicle.CanCharge() && h.energy != nil {
		if stats, err = h.energy.PlaceStats(ctx, vehicle.ID, vehicle.UserID, period, time.Now()); err != nil {
			return nil, err
		}
		if locations, err = h.repo.ListChargingLocations(ctx, vehicle.ID, vehicle.UserID); err != nil {
			return nil, err
		}
	}
	out := buildIntegrationPlaces(vehicle.ID, vehicle.Currency, months, stats, locations)
	return &out, nil
}

// GetChargingPlaces serves ChargingPlaces.
func (h *IntegrationReadHandler) GetChargingPlaces(w http.ResponseWriter, r *http.Request) {
	months := integrationPlacesDefaultMonths
	if raw := r.URL.Query().Get("months"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_months", "months must be between 0 and 240"))
			return
		}
		months = n
	}
	out, err := h.ChargingPlaces(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), months)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the charging places")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
