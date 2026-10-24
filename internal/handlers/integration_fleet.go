package handlers

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// errIntegrationUnavailable is a view whose service the handler was built without.
var errIntegrationUnavailable = errors.New("this view is not available")

// integrationFleetMonths is how many months of the household cost the fleet view carries.
const integrationFleetMonths = 12

// IntegrationFleetTotal is what the vehicles of one currency cost in the month in progress. Currencies are never
// added together.
type IntegrationFleetTotal struct {
	Currency         string            `json:"currency" jsonschema:"currency of the vehicles counted"`
	Vehicles         int               `json:"vehicles" jsonschema:"number of vehicles in that currency"`
	CurrentMonthCost IntegrationAmount `json:"current_month_cost" jsonschema:"what they cost so far in the month in progress"`
}

// IntegrationFleetBudget is the household's monthly spending target and where the month stands against it.
type IntegrationFleetBudget struct {
	Monthly   IntegrationAmount `json:"monthly" jsonschema:"the monthly target the owner set"`
	Spent     IntegrationAmount `json:"spent_this_month" jsonschema:"what the vehicles in the budget's currency cost so far this month"`
	Remaining IntegrationAmount `json:"remaining" jsonschema:"target minus spent, negative once the target is exceeded"`
	UsedPct   float64           `json:"used_pct" jsonschema:"spent as a percentage of the target"`
}

// IntegrationFleetVehicle compares one vehicle with the others of the account.
type IntegrationFleetVehicle struct {
	VehicleID        string              `json:"vehicle_id" jsonschema:"identifier to pass to the other tools"`
	Name             string              `json:"name" jsonschema:"name the owner gave the vehicle, user text"`
	Make             string              `json:"make" jsonschema:"make as the owner typed it, user text; may be empty"`
	Model            string              `json:"model" jsonschema:"model as the owner typed it, user text; may be empty"`
	Powertrain       string              `json:"powertrain" jsonschema:"EV, ICE, PHEV or REEV"`
	Currency         string              `json:"currency" jsonschema:"currency of the vehicle's amounts"`
	Odometer         IntegrationDistance `json:"odometer" jsonschema:"current odometer reading"`
	MonthCost        IntegrationAmount   `json:"current_month_cost" jsonschema:"cost so far in the month in progress"`
	MonthDistance    IntegrationDistance `json:"current_month_distance" jsonschema:"distance driven so far in the month in progress"`
	RunningCostPerKm *IntegrationAmount  `json:"running_cost_per_km" jsonschema:"what it costs to run per kilometre (energy, upkeep, insurance, financing); null while the distance is unknown"`
	FullCostPerKm    *IntegrationAmount  `json:"full_cost_per_km" jsonschema:"running cost plus depreciation and amortized tires, per kilometre; null while the distance is unknown"`
	AnnualCost       *IntegrationAmount  `json:"annual_cost" jsonschema:"full cost extrapolated to twelve months from the observed history; null under three months of history"`
	EnergyCostPer100 *IntegrationAmount  `json:"energy_cost_per_100km" jsonschema:"energy cost per 100 km; null when unknown, 0 is a known free energy"`
	KwhPer100km      *float64            `json:"kwh_per_100km" jsonschema:"electric consumption per 100 km; null when the vehicle used no electricity"`
	LitresPer100km   *float64            `json:"litres_per_100km" jsonschema:"fuel consumption per 100 km; null when the vehicle used no fuel"`
	CompletenessPct  int                 `json:"completeness_pct" jsonschema:"completeness score of the vehicle's records, from 0 to 100"`
	Comparable       bool                `json:"comparable" jsonschema:"true when the vehicle can be ranked: enough distance, a known depreciation and enough complete records; otherwise do not use it to say which vehicle is cheapest"`
	CostRank         *int                `json:"full_cost_rank" jsonschema:"1 for the cheapest full cost per km among the comparable vehicles of the same currency; null when the vehicle is not comparable"`
}

// IntegrationFleetMonth is the household cost of one month, for the vehicles in the fleet currency.
type IntegrationFleetMonth struct {
	Month    string              `json:"month" jsonschema:"calendar month, YYYY-MM"`
	Total    IntegrationAmount   `json:"total" jsonschema:"what the vehicles in the fleet currency cost"`
	Energy   IntegrationAmount   `json:"energy" jsonschema:"their energy cost"`
	Other    IntegrationAmount   `json:"other" jsonschema:"everything else"`
	Distance IntegrationDistance `json:"distance" jsonschema:"distance driven by all the vehicles, whatever their currency"`
}

// IntegrationFleetDriver is the share of the fleet's distance one person drove.
type IntegrationFleetDriver struct {
	Name       string              `json:"name" jsonschema:"name of the driver as entered in the account, user text"`
	Distance   IntegrationDistance `json:"distance" jsonschema:"distance this person drove"`
	Percentage float64             `json:"percentage" jsonschema:"share of the distance of the drives that name a driver"`
}

// IntegrationFleet is the household view: every vehicle of the account side by side.
type IntegrationFleet struct {
	VehicleCount  int                       `json:"vehicle_count" jsonschema:"number of vehicles in the account"`
	CurrentMonth  string                    `json:"current_month" jsonschema:"the month in progress, YYYY-MM"`
	FleetCurrency string                    `json:"fleet_currency" jsonschema:"currency of monthly_costs: the one of the first vehicle; vehicles in another currency are left out of those sums"`
	Totals        []IntegrationFleetTotal   `json:"current_month_totals" jsonschema:"the month in progress, one line per currency"`
	Budget        *IntegrationFleetBudget   `json:"budget" jsonschema:"the monthly target; null when none is set or no vehicle uses its currency"`
	Vehicles      []IntegrationFleetVehicle `json:"vehicles" jsonschema:"the vehicles with their comparison figures"`
	MonthlyCosts  []IntegrationFleetMonth   `json:"monthly_costs" jsonschema:"the last 12 months of the household cost, oldest first"`
	Drivers       []IntegrationFleetDriver  `json:"drivers" jsonschema:"who drove how much, by name; empty when no drive names its driver"`
}

func fleetAmount(c *money.Cents, cur string) *IntegrationAmount { return optionalAmount(c, cur) }

func buildIntegrationFleet(res *models.FleetSummaryResponse) IntegrationFleet {
	out := IntegrationFleet{
		VehicleCount: res.TotalVehicles, CurrentMonth: res.CurrentMonth, FleetCurrency: res.Currency,
		Totals: []IntegrationFleetTotal{}, Vehicles: []IntegrationFleetVehicle{}, MonthlyCosts: []IntegrationFleetMonth{}, Drivers: []IntegrationFleetDriver{},
	}
	spentIn := map[string]money.Cents{}
	for _, t := range res.Totals {
		spentIn[t.Currency] = t.CurrentMonthCost
		out.Totals = append(out.Totals, IntegrationFleetTotal{Currency: t.Currency, Vehicles: t.VehicleCount, CurrentMonthCost: amountOf(t.CurrentMonthCost, t.Currency)})
	}
	if res.MonthlyBudget != nil && res.BudgetCurrency != nil {
		if spent, ok := spentIn[*res.BudgetCurrency]; ok {
			cur := *res.BudgetCurrency
			b := &IntegrationFleetBudget{Monthly: amountOf(*res.MonthlyBudget, cur), Spent: amountOf(spent, cur), Remaining: amountOf(*res.MonthlyBudget-spent, cur)}
			if *res.MonthlyBudget > 0 {
				b.UsedPct = roundTo(spent.Float()/res.MonthlyBudget.Float()*100, 1)
			}
			out.Budget = b
		}
	}

	rank := map[string]int{}
	byCurrency := map[string][]models.VehicleFleetMetric{}
	for _, v := range res.Vehicles {
		if v.Comparable && v.FullCostPerKm != nil {
			byCurrency[v.Currency] = append(byCurrency[v.Currency], v)
		}
	}
	for _, group := range byCurrency {
		sort.SliceStable(group, func(i, j int) bool { return *group[i].FullCostPerKm < *group[j].FullCostPerKm })
		for i, v := range group {
			rank[v.VehicleID] = i + 1
		}
	}
	for _, v := range res.Vehicles {
		item := IntegrationFleetVehicle{
			VehicleID: v.VehicleID, Name: v.Name, Make: v.Make, Model: v.Model, Powertrain: v.Powertrain, Currency: v.Currency,
			Odometer: kilometres(v.CurrentOdometer), MonthCost: amountOf(v.MonthCost, v.Currency), MonthDistance: kilometres(v.MonthDistanceKm),
			RunningCostPerKm: ratePerKm(v.RunningCostPerKm, v.Currency), FullCostPerKm: ratePerKm(v.FullCostPerKm, v.Currency),
			AnnualCost: fleetAmount(v.AnnualCost, v.Currency), EnergyCostPer100: decimalAmount(v.EnergyCostPer100Km, v.Currency, 2),
			KwhPer100km: v.KwhPer100Km, LitresPer100km: v.LitersPer100Km, CompletenessPct: v.CompletenessPct, Comparable: v.Comparable,
		}
		if r, ok := rank[v.VehicleID]; ok {
			item.CostRank = &r
		}
		out.Vehicles = append(out.Vehicles, item)
	}

	months := res.MonthlyCosts
	if len(months) > integrationFleetMonths {
		months = months[len(months)-integrationFleetMonths:]
	}
	for _, m := range months {
		out.MonthlyCosts = append(out.MonthlyCosts, IntegrationFleetMonth{
			Month: m.Month, Total: amountOf(m.TotalCost, res.Currency), Energy: amountOf(m.EnergyCost, res.Currency),
			Other: amountOf(m.OtherCost, res.Currency), Distance: kilometres(m.DistanceKm),
		})
	}
	for _, d := range res.MemberKmShares {
		out.Drivers = append(out.Drivers, IntegrationFleetDriver{Name: d.DisplayName, Distance: kilometres(d.DistanceKm), Percentage: d.Percentage})
	}
	return out
}

// WithFleet gives the fleet view its service.
func (h *IntegrationReadHandler) WithFleet(s *services.FleetService) *IntegrationReadHandler {
	h.fleet = s
	return h
}

// FleetSummary compares the vehicles of the account.
func (h *IntegrationReadHandler) FleetSummary(ctx context.Context, userID string) (*IntegrationFleet, error) {
	if h.fleet == nil {
		return nil, errIntegrationUnavailable
	}
	if _, limited := middleware.VehicleScope(ctx); limited {
		return nil, errIntegrationVehiclesLimited
	}
	res, err := h.fleet.GetSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationFleet(res)
	return &out, nil
}

// GetFleetSummary serves FleetSummary.
func (h *IntegrationReadHandler) GetFleetSummary(w http.ResponseWriter, r *http.Request) {
	out, err := h.FleetSummary(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the fleet summary")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
