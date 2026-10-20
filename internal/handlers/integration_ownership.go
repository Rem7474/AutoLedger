package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationEconomicCost is what owning the vehicle costs once the purchase is counted: the running costs with the
// tires spread over their life, plus what the vehicle lost in value.
type IntegrationEconomicCost struct {
	FullCost          IntegrationAmount   `json:"full_cost" jsonschema:"economic cost of ownership: running costs with the tires amortized per km, plus the depreciation (or the financing as a whole for a lease); unlike the cash costs of get_vehicle_costs it accounts for the purchase"`
	FullCostPerKm     *IntegrationAmount  `json:"full_cost_per_km" jsonschema:"full cost per kilometre; null while the distance is unknown"`
	Depreciation      IntegrationAmount   `json:"depreciation" jsonschema:"value the vehicle lost: purchase price and fees, net of incentives, minus the resale value; 0 when it cannot be computed (see depreciation_known) and for a lease"`
	DepreciationKnown bool                `json:"depreciation_known" jsonschema:"false when a bought vehicle lacks its purchase price, resale value or holding period: the depreciation is then 0 and the full cost is understated, so do not compare it with a complete one"`
	DepreciationPerKm *IntegrationAmount  `json:"depreciation_per_km" jsonschema:"depreciation per kilometre; null while the distance is unknown"`
	Distance          IntegrationDistance `json:"distance" jsonschema:"distance the per-km figures divide by: the largest of the tracked drives, their odometer span and the distance since acquisition"`
	TiresPaid         IntegrationAmount   `json:"tires_paid" jsonschema:"what the tires cost in cash, at purchase; the full cost counts tires_amortized instead"`
	TiresAmortized    IntegrationAmount   `json:"tires_amortized" jsonschema:"the tires' cost spread over the distance they were used"`
	CarpoolRevenue    IntegrationAmount   `json:"carpool_revenue" jsonschema:"what passengers paid the driver for carpooling"`
	FullCostNet       IntegrationAmount   `json:"full_cost_net" jsonschema:"full cost minus the carpool revenue"`
	FullCostNetPerKm  *IntegrationAmount  `json:"full_cost_net_per_km" jsonschema:"net full cost per kilometre; null while the distance is unknown"`
}

// IntegrationContract is the lease or loan of a vehicle, with its mileage allowance used so far.
type IntegrationContract struct {
	Start               *string             `json:"start" jsonschema:"day the contract started, YYYY-MM-DD"`
	End                 *string             `json:"end" jsonschema:"day the contract ends, YYYY-MM-DD; null when its duration is not entered"`
	DurationMonths      *int                `json:"duration_months" jsonschema:"duration of the contract in months; null when not entered"`
	MonthlyRent         *IntegrationAmount  `json:"monthly_rent" jsonschema:"monthly instalment of a lease; null when not entered or not a lease"`
	DownPayment         *IntegrationAmount  `json:"down_payment" jsonschema:"first rent or down payment of a lease; null when none"`
	IncludesMaintenance bool                `json:"includes_maintenance" jsonschema:"true when the lease pays for the maintenance, so it is not a cost to the owner"`
	IncludesInsurance   bool                `json:"includes_insurance" jsonschema:"true when the lease includes the insurance"`
	IncludesTires       bool                `json:"includes_tires" jsonschema:"true when the lease includes the tires"`
	PurchaseOption      *IntegrationAmount  `json:"purchase_option" jsonschema:"price to buy the vehicle at the end of a LOA; null when none"`
	OptionExercisedOn   *string             `json:"option_exercised_on" jsonschema:"day the purchase option was exercised, YYYY-MM-DD; null when it was not"`
	Mileage             *IntegrationMileage `json:"mileage" jsonschema:"the mileage allowance and where the vehicle stands against it; null when the contract has no mileage allowance"`
}

// IntegrationMileage compares the distance driven under a lease with what the contract allows.
type IntegrationMileage struct {
	AllowancePerYear *IntegrationDistance `json:"allowance_per_year" jsonschema:"distance allowed per year; null when only a total is entered"`
	AllowanceTotal   *IntegrationDistance `json:"allowance_total" jsonschema:"distance allowed over the whole contract"`
	AllowanceToDate  IntegrationDistance  `json:"allowance_to_date" jsonschema:"the part of the allowance that has elapsed, pro rata of the time"`
	Driven           IntegrationDistance  `json:"driven" jsonschema:"distance driven since the start of the contract"`
	ExcessPrice      *IntegrationAmount   `json:"excess_price_per_km" jsonschema:"price charged per kilometre above the allowance; null when not entered"`
	ExcessAccrued    IntegrationAmount    `json:"excess_accrued" jsonschema:"cost of the kilometres above the pro-rata allowance so far"`
	ExcessProjected  IntegrationAmount    `json:"excess_projected" jsonschema:"cost of the excess expected at the end of the contract if the pace stays the same"`
}

// IntegrationResale is the estimated value of the vehicle today.
type IntegrationResale struct {
	CurrentValue  IntegrationAmount  `json:"current_value" jsonschema:"estimated value today, from the owner's purchase price and expected resale value, adjusted for age, distance and battery health"`
	StartValue    IntegrationAmount  `json:"start_value" jsonschema:"value at the start of the ownership (or of the lease contract)"`
	ExpectedValue IntegrationAmount  `json:"expected_value" jsonschema:"value expected at the end of the holding period (or of the lease contract)"`
	HoldingMonths int                `json:"holding_months" jsonschema:"planned holding period in months"`
	AgeMonths     float64            `json:"age_months" jsonschema:"months since the start of the ownership"`
	Progress      float64            `json:"progress" jsonschema:"share of the planned depreciation already consumed: 0 is new, 1 is the planned resale date"`
	HealthPercent *float64           `json:"battery_health_percent" jsonschema:"battery state of health used in the estimate; null when unknown"`
	Depreciation  IntegrationAmount  `json:"depreciation_to_date" jsonschema:"start value minus the current value"`
	UnderLease    bool               `json:"under_lease" jsonschema:"true while a LOA is running: the values are then those of the vehicle itself, not what the owner paid"`
	OptionMargin  *IntegrationAmount `json:"purchase_option_margin" jsonschema:"for a LOA, the expected value at the end of the contract minus the purchase option price: positive when buying is worth it; null otherwise"`
	Assumptions   string             `json:"assumptions" jsonschema:"how the estimate is made"`
}

// IntegrationOwnership is the part of a vehicle's cost that the cash costs leave out: the purchase, the contract and
// the resale value.
type IntegrationOwnership struct {
	VehicleID         string                  `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency          string                  `json:"currency" jsonschema:"currency of every amount"`
	AcquisitionType   *string                 `json:"acquisition_type" jsonschema:"CASH (bought outright), LOAN, LOA (lease with a purchase option) or LLD (long-term rental); null when the acquisition is not entered"`
	Acquisition       IntegrationAmount       `json:"acquisition_cost" jsonschema:"purchase price and fees net of incentives; 0 for a lease still running"`
	Economic          IntegrationEconomicCost `json:"economic_cost" jsonschema:"what owning the vehicle costs, purchase included"`
	Contract          *IntegrationContract    `json:"contract" jsonschema:"the lease or loan; null for a vehicle bought outright"`
	Resale            *IntegrationResale      `json:"resale" jsonschema:"the estimated value of the vehicle today; null when it cannot be estimated, see resale_unavailable"`
	ResaleUnavailable *string                 `json:"resale_unavailable" jsonschema:"why there is no resale estimate; null when there is one"`
}

// buildIntegrationOwnership shapes the economic figures of the TCO, the contract it carries and the optional
// residual-value projection (nil when it could not be made, then unavailable says why).
func buildIntegrationOwnership(vehicle *models.Vehicle, sum *services.TCOSummary, residual *models.ResidualValue, unavailable string) IntegrationOwnership {
	cur := vehicle.Currency
	out := IntegrationOwnership{
		VehicleID:   vehicle.ID,
		Currency:    cur,
		Acquisition: amountOf(sum.AcquisitionCost, cur),
		Economic: IntegrationEconomicCost{
			FullCost:          amountOf(sum.FullCost, cur),
			FullCostPerKm:     ratePerKm(sum.FullCostPerKm, cur),
			Depreciation:      amountOf(sum.DepreciationCost, cur),
			DepreciationKnown: sum.DepreciationKnown,
			DepreciationPerKm: ratePerKm(sum.DepreciationCostPerKm, cur),
			Distance:          kilometres(sum.DistanceBasisKm),
			TiresPaid:         amountOf(sum.TiresCost, cur),
			TiresAmortized:    amountOf(sum.TiresAmortizedCost, cur),
			CarpoolRevenue:    amountOf(sum.CarpoolRevenue, cur),
			FullCostNet:       amountOf(sum.FullCostNet, cur),
			FullCostNetPerKm:  ratePerKm(sum.FullCostNetPerKm, cur),
		},
	}
	if sum.AcquisitionType != "" {
		t := sum.AcquisitionType
		out.AcquisitionType = &t
	}
	if sum.AcquisitionType == models.AcquisitionLOA || sum.AcquisitionType == models.AcquisitionLLD || sum.AcquisitionType == models.AcquisitionLoan {
		out.Contract = buildIntegrationContract(cur, sum)
	}
	if residual != nil {
		out.Resale = buildIntegrationResale(cur, residual)
	} else if unavailable != "" {
		out.ResaleUnavailable = &unavailable
	}
	return out
}

func optionalAmount(c *money.Cents, cur string) *IntegrationAmount {
	if c == nil {
		return nil
	}
	a := amountOf(*c, cur)
	return &a
}

func buildIntegrationContract(cur string, sum *services.TCOSummary) *IntegrationContract {
	c := &IntegrationContract{
		Start:               reminderDay(sum.ContractStartDate),
		End:                 reminderDay(sum.ContractEndDate),
		DurationMonths:      sum.ContractDurationMonths,
		MonthlyRent:         optionalAmount(sum.LeaseMonthlyRent, cur),
		DownPayment:         optionalAmount(sum.LeaseDownPayment, cur),
		IncludesMaintenance: sum.LeaseIncludesMaintenance,
		IncludesInsurance:   sum.LeaseIncludesInsurance,
		IncludesTires:       sum.LeaseIncludesTires,
		PurchaseOption:      optionalAmount(sum.LeasePurchaseOptionPrice, cur),
		OptionExercisedOn:   reminderDay(sum.OptionExercisedDate),
	}
	if sum.LeaseKmAllowanceTotal != nil || sum.LeaseKmAllowancePerYear != nil {
		c.Mileage = &IntegrationMileage{
			AllowancePerYear: reminderKm(sum.LeaseKmAllowancePerYear),
			AllowanceTotal:   reminderKm(sum.LeaseKmAllowanceTotal),
			AllowanceToDate:  kilometres(sum.LeaseKmAllowanceToDate),
			Driven:           kilometres(sum.LeaseKmDriven),
			ExcessPrice:      ratePerKm(sum.LeaseExcessKmPrice, cur),
			ExcessAccrued:    amountOf(sum.LeaseExcessKmCost, cur),
			ExcessProjected:  amountOf(sum.LeaseExcessKmProjected, cur),
		}
	}
	return c
}

func buildIntegrationResale(cur string, r *models.ResidualValue) *IntegrationResale {
	out := &IntegrationResale{
		CurrentValue:  amountOf(r.CurrentValue, cur),
		StartValue:    amountOf(r.PurchasePrice, cur),
		ExpectedValue: amountOf(r.ExpectedValue, cur),
		HoldingMonths: r.HoldingMonths,
		AgeMonths:     r.AgeMonths,
		Progress:      r.Progress,
		HealthPercent: r.HealthPercent,
		Depreciation:  amountOf(r.Depreciation, cur),
		UnderLease:    r.Lease,
		OptionMargin:  optionalAmount(r.OptionMargin, cur),
		Assumptions:   "Value falls geometrically from the start value to the expected value over the holding period, by age only, then is scaled by the battery state of health when known.",
	}
	return out
}

// WithResidual lets the ownership view estimate the resale value; without it the estimate is left out.
func (h *IntegrationReadHandler) WithResidual(s *services.ResidualService) *IntegrationReadHandler {
	h.residual = s
	return h
}

// Ownership gives the economic cost, the contract and the resale value of a vehicle.
func (h *IntegrationReadHandler) Ownership(ctx context.Context, userID, vehicleID string) (*IntegrationOwnership, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	var residual *models.ResidualValue
	unavailable := ""
	if h.residual != nil {
		residual, err = h.residual.Residual(ctx, vehicle, services.ResidualOptions{KmShare: 0.5, HealthWeight: 1})
		if err != nil {
			apiErr, ok := apierror.As(err)
			if !ok {
				return nil, err
			}
			residual, unavailable = nil, apiErr.Message
		}
	}
	out := buildIntegrationOwnership(vehicle, sum, residual, unavailable)
	return &out, nil
}

// GetOwnership serves Ownership.
func (h *IntegrationReadHandler) GetOwnership(w http.ResponseWriter, r *http.Request) {
	out, err := h.Ownership(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the ownership figures")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
