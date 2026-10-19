package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// integrationComparisonsMax caps the scenarios listed.
const integrationComparisonsMax = 50

// WithComparison gives the comparison view the service that evaluates a saved scenario.
func (h *IntegrationReadHandler) WithComparison(s *services.ComparisonService) *IntegrationReadHandler {
	h.comparison = s
	return h
}

// IntegrationComparisonScenario is a saved electric-versus-combustion scenario, without its result.
type IntegrationComparisonScenario struct {
	ID        string              `json:"id" jsonschema:"identifier to pass as scenario to get the result"`
	Name      string              `json:"name" jsonschema:"name the owner gave the scenario, user text"`
	Mode      string              `json:"mode" jsonschema:"RETROSPECTIVE when the tracked vehicle's side comes from its recorded costs, PROJECTION when the owner described it"`
	Vehicle   *string             `json:"vehicle" jsonschema:"name of the tracked vehicle of a RETROSPECTIVE scenario, user text; null for a PROJECTION or when the account cannot see that vehicle any more"`
	Currency  string              `json:"currency" jsonschema:"ISO 4217 currency code the scenario is computed in"`
	AnnualKm  IntegrationDistance `json:"annual_distance" jsonschema:"distance driven in a year in the scenario"`
	Years     int                 `json:"years" jsonschema:"length of the compared period, in years"`
	FuelType  string              `json:"fuel_type" jsonschema:"fuel of the combustion vehicle it is compared with: SP95_E10, SP98, DIESEL, E85 or GPL"`
	LPer100Km float64             `json:"l_per_100km" jsonschema:"consumption of the combustion vehicle, litres per 100 km"`
	UpdatedAt string              `json:"updated_at" jsonschema:"when the scenario was last saved, RFC 3339"`
}

// IntegrationComparisonSide is the cost of one vehicle over the compared period.
type IntegrationComparisonSide struct {
	Energy       IntegrationAmount  `json:"energy" jsonschema:"electricity or fuel over the period"`
	Maintenance  IntegrationAmount  `json:"maintenance" jsonschema:"maintenance over the period"`
	Insurance    IntegrationAmount  `json:"insurance" jsonschema:"insurance over the period"`
	Tax          IntegrationAmount  `json:"tax" jsonschema:"yearly taxes over the period"`
	Depreciation IntegrationAmount  `json:"depreciation" jsonschema:"purchase price minus the resale value at the end of the period"`
	Total        IntegrationAmount  `json:"total" jsonschema:"all of the above"`
	PerYear      IntegrationAmount  `json:"per_year" jsonschema:"total over the number of years"`
	PerMonth     IntegrationAmount  `json:"per_month" jsonschema:"total over the number of months"`
	CostPerKm    *IntegrationAmount `json:"cost_per_km" jsonschema:"total over the distance of the period; null when it is unknown"`
}

// IntegrationComparisonSensitivity is the saving when one assumption varies.
type IntegrationComparisonSensitivity struct {
	Assumption string            `json:"assumption" jsonschema:"the assumption that varies, in words"`
	Savings    IntegrationAmount `json:"savings" jsonschema:"saving of the tracked vehicle with that assumption; negative when it costs more"`
	Shift      IntegrationAmount `json:"shift" jsonschema:"difference with the saving of the scenario"`
}

// IntegrationComparison is the outcome of a saved scenario.
type IntegrationComparison struct {
	ScenarioID        string                             `json:"scenario_id" jsonschema:"the scenario this result belongs to"`
	Tracked           IntegrationComparisonSide          `json:"tracked" jsonschema:"the electric or plug-in vehicle the owner tracks"`
	Combustion        IntegrationComparisonSide          `json:"combustion" jsonschema:"the combustion vehicle it is compared with"`
	Savings           *IntegrationAmount                 `json:"savings" jsonschema:"what the tracked vehicle saves over the period compared with the combustion one, negative when it costs more; null while the tracked vehicle's depreciation is unknown (see depreciation_known), because its total would be incomplete"`
	DepreciationKnown bool                               `json:"depreciation_known" jsonschema:"false when the tracked vehicle has no depreciation entered, so its total misses it"`
	BreakEvenYears    *float64                           `json:"break_even_years" jsonschema:"years until the cumulated cash outlay of the tracked vehicle drops below the combustion one; null when it never does, see break_even_reason"`
	BreakEvenReason   string                             `json:"break_even_reason" jsonschema:"why break_even_years is null, empty when it is reached"`
	BreakEvenNetYears *float64                           `json:"break_even_net_years" jsonschema:"the same break-even once each vehicle is credited with its resale value at that point; null when it is not reached"`
	Sensitivity       []IntegrationComparisonSensitivity `json:"sensitivity" jsonschema:"how the saving changes when an assumption varies"`
	Assumptions       []string                           `json:"assumptions" jsonschema:"what the figures rest on and what was estimated, in words"`
}

// IntegrationComparisons lists the saved scenarios and, for one of them, the result.
type IntegrationComparisons struct {
	Scenarios []IntegrationComparisonScenario `json:"scenarios" jsonschema:"the scenarios saved on the account, most recent first, 50 at most"`
	Result    *IntegrationComparison          `json:"result" jsonschema:"the result of the scenario asked for; null when none was asked for"`
}

func comparisonSide(s services.CostSide, currency string) IntegrationComparisonSide {
	out := IntegrationComparisonSide{
		Energy: amountOf(s.Energy, currency), Maintenance: amountOf(s.Maintenance, currency), Insurance: amountOf(s.Insurance, currency),
		Tax: amountOf(s.Tax, currency), Depreciation: amountOf(s.Depreciation, currency), Total: amountOf(s.Total, currency),
		PerYear: amountOf(s.PerYear, currency), PerMonth: amountOf(s.PerMonth, currency),
	}
	if s.CostPerKm > 0 {
		out.CostPerKm = &IntegrationAmount{Amount: strconv.FormatFloat(s.CostPerKm, 'f', 4, 64), Currency: currency}
	}
	return out
}

func buildIntegrationComparison(scenarioID, currency string, res *services.ComparisonResult) IntegrationComparison {
	out := IntegrationComparison{
		ScenarioID: scenarioID, Tracked: comparisonSide(res.Tracked, currency), Combustion: comparisonSide(res.ICE, currency),
		DepreciationKnown: res.TrackedDepreciationKnown, BreakEvenYears: res.BreakEvenYear, BreakEvenReason: res.BreakEvenReason,
		BreakEvenNetYears: res.BreakEvenNetYear, Sensitivity: []IntegrationComparisonSensitivity{}, Assumptions: []string{},
	}
	if res.TrackedDepreciationKnown {
		savings := amountOf(res.TrackedSavings, currency)
		out.Savings = &savings
	}
	for _, row := range res.Sensitivity {
		label := ""
		if row.Label != nil {
			label = row.Label.Error()
		}
		out.Sensitivity = append(out.Sensitivity, IntegrationComparisonSensitivity{Assumption: label, Savings: amountOf(row.TrackedSavings, currency), Shift: amountOf(row.DeltaShift, currency)})
	}
	for _, a := range res.Assumptions {
		if a != nil {
			out.Assumptions = append(out.Assumptions, a.Error())
		}
	}
	return out
}

func buildIntegrationComparisonScenario(sc models.ComparisonScenario, vehicleName *string) IntegrationComparisonScenario {
	return IntegrationComparisonScenario{
		ID: sc.ID, Name: sc.Name, Mode: sc.Mode, Vehicle: vehicleName, Currency: sc.Currency, AnnualKm: kilometres(sc.AnnualKm), Years: sc.Years,
		FuelType: sc.ICE.FuelType, LPer100Km: sc.ICE.LPer100Km, UpdatedAt: sc.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// Comparisons lists the saved scenarios and, when scenarioID is not empty, evaluates that one. Nothing is written.
func (h *IntegrationReadHandler) Comparisons(ctx context.Context, userID, scenarioID string) (*IntegrationComparisons, error) {
	if h.comparison == nil {
		return nil, apierror.New("internal", "Comparisons are unavailable")
	}
	scenarios, err := h.repo.ListComparisonScenarios(ctx, userID)
	if err != nil {
		return nil, err
	}
	vehicles, err := h.repo.ListVehiclesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	currencies := map[string]string{}
	for _, v := range vehicles {
		names[v.ID] = v.Name
		currencies[v.ID] = v.Currency
	}
	out := &IntegrationComparisons{Scenarios: []IntegrationComparisonScenario{}}
	var chosen *models.ComparisonScenario
	for i := range scenarios {
		sc := scenarios[i]
		var name *string
		if sc.VehicleID != nil {
			if n, ok := names[*sc.VehicleID]; ok {
				name = &n
			}
			if c, ok := currencies[*sc.VehicleID]; ok && sc.Currency == "" {
				sc.Currency = c
			}
		}
		if sc.ID == scenarioID {
			chosen = &sc
		}
		if len(out.Scenarios) < integrationComparisonsMax {
			out.Scenarios = append(out.Scenarios, buildIntegrationComparisonScenario(sc, name))
		}
	}
	if scenarioID == "" {
		return out, nil
	}
	if chosen == nil {
		return nil, errIntegrationComparisonNotFound
	}
	if chosen.VehicleID != nil {
		if _, err := h.vehicleOf(ctx, userID, *chosen.VehicleID); err != nil {
			return nil, err
		}
	}
	res, err := h.comparison.Compare(ctx, chosen)
	if err != nil {
		return nil, err
	}
	result := buildIntegrationComparison(chosen.ID, chosen.Currency, res)
	out.Result = &result
	return out, nil
}

// GetComparisons serves Comparisons. The scenario to evaluate is the optional `scenario` query parameter.
func (h *IntegrationReadHandler) GetComparisons(w http.ResponseWriter, r *http.Request) {
	out, err := h.Comparisons(r.Context(), middleware.GetUserID(r.Context()), strings.TrimSpace(r.URL.Query().Get("scenario")))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the comparison")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
