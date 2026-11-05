package handlers

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// Sections of the summary that cost more than its default figures and are only computed when asked for.
const (
	summarySectionTires     = "tires"
	summarySectionBattery   = "battery"
	summarySectionOwnership = "ownership"
	summarySectionSources   = "sources"
)

var summarySections = []string{summarySectionTires, summarySectionBattery, summarySectionOwnership, summarySectionSources}

// SummaryInclude is the set of optional sections a summary request asks for.
type SummaryInclude struct {
	Tires, Battery, Ownership, Sources bool
}

// parseSummaryInclude reads the include query value: a comma-separated list of section names. An empty value asks
// for none; a name that is not a section is refused rather than ignored, so a typo does not silently drop a figure.
func parseSummaryInclude(raw string) (SummaryInclude, *apierror.Error) {
	var inc SummaryInclude
	for _, part := range strings.Split(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		switch name {
		case "":
		case summarySectionTires:
			inc.Tires = true
		case summarySectionBattery:
			inc.Battery = true
		case summarySectionOwnership:
			inc.Ownership = true
		case summarySectionSources:
			inc.Sources = true
		default:
			return SummaryInclude{}, apierror.New("request.invalid_value", "include accepts: "+strings.Join(summarySections, ", "))
		}
	}
	return inc, nil
}

// IntegrationSummaryTires is how worn the tires on the car are, the worst of them for each figure.
type IntegrationSummaryTires struct {
	Mounted                int                  `json:"mounted" jsonschema:"number of tires mounted on the vehicle, those in storage or disposed of are left out of every figure"`
	WorstWearPct           float64              `json:"worst_wear_pct" jsonschema:"highest wear of the mounted tires, in percent of the usable tread"`
	WorstLifeUsedPct       float64              `json:"worst_life_used_pct" jsonschema:"highest share of the expected lifespan already driven, in percent; can exceed 100"`
	SmallestRemaining      *IntegrationDistance `json:"smallest_remaining_distance" jsonschema:"smallest distance left before a mounted tire reaches the legal minimum"`
	NearestReplacementDate *string              `json:"nearest_replacement_date" jsonschema:"earliest expected replacement day of the mounted tires, YYYY-MM-DD; null when none can be predicted"`
	WorstCondition         string               `json:"worst_condition" jsonschema:"GOOD, WARNING or CRITICAL, the worst of the mounted tires"`
}

// IntegrationSummaryOwnership is the part of the ownership figures a dashboard shows.
type IntegrationSummaryOwnership struct {
	FullCostPerKm     *IntegrationAmount  `json:"full_cost_per_km" jsonschema:"economic cost per kilometre, the purchase included; null while the distance is unknown"`
	CurrentValue      *IntegrationAmount  `json:"current_value" jsonschema:"estimated value of the vehicle today; null when it cannot be estimated"`
	DepreciationKnown bool                `json:"depreciation_known" jsonschema:"false when the depreciation could not be computed, so full_cost_per_km is understated"`
	Mileage           *IntegrationMileage `json:"mileage" jsonschema:"the mileage allowance of a lease and where the vehicle stands against it; null without an allowance"`
}

// IntegrationSummarySources is how fresh a vehicle's data is.
type IntegrationSummarySources struct {
	TeslaMateConnected bool    `json:"teslamate_connected" jsonschema:"true when the vehicle is connected to a TeslaMate instance; only the owner of the vehicle sees it as true"`
	LastRecord         *string `json:"last_record" jsonschema:"RFC 3339 instant of the most recent record any origin wrote; null when the vehicle has none"`
	LastSync           *string `json:"last_sync" jsonschema:"RFC 3339 instant of the last successful TeslaMate synchronization; null when none has succeeded"`
	PendingCharges     int     `json:"pending_charges" jsonschema:"charging sessions pushed to the account that no vehicle claimed yet, all vehicles together"`
}

// summaryTires gives the wear of the mounted tires; nil when the vehicle has none.
func summaryTires(vehicleID, currency string, stats []services.TireWearStats) *IntegrationSummaryTires {
	var out *IntegrationSummaryTires
	for _, tire := range buildIntegrationTires(vehicleID, currency, stats).Tires {
		if !slices.Contains([]string{string(models.TirePosFL), string(models.TirePosFR), string(models.TirePosRL), string(models.TirePosRR)}, tire.Position) {
			continue
		}
		if out == nil {
			out = &IntegrationSummaryTires{WorstCondition: tire.Condition}
		}
		out.Mounted++
		out.WorstWearPct = max(out.WorstWearPct, tire.WearPct)
		out.WorstLifeUsedPct = max(out.WorstLifeUsedPct, tire.LifeUsedPct)
		if tire.RemainingKm != nil && (out.SmallestRemaining == nil || tire.RemainingKm.Value < out.SmallestRemaining.Value) {
			remaining := *tire.RemainingKm
			out.SmallestRemaining = &remaining
		}
		if tire.ReplacementDate != nil && (out.NearestReplacementDate == nil || *tire.ReplacementDate < *out.NearestReplacementDate) {
			date := *tire.ReplacementDate
			out.NearestReplacementDate = &date
		}
		if conditionRank(tire.Condition) > conditionRank(out.WorstCondition) {
			out.WorstCondition = tire.Condition
		}
	}
	return out
}

func conditionRank(condition string) int {
	switch condition {
	case "CRITICAL":
		return 2
	case "WARNING":
		return 1
	}
	return 0
}

func summaryOwnership(o IntegrationOwnership) *IntegrationSummaryOwnership {
	out := &IntegrationSummaryOwnership{FullCostPerKm: o.Economic.FullCostPerKm, DepreciationKnown: o.Economic.DepreciationKnown}
	if o.Resale != nil {
		value := o.Resale.CurrentValue
		out.CurrentValue = &value
	}
	if o.Contract != nil {
		out.Mileage = o.Contract.Mileage
	}
	return out
}

func summarySources(s IntegrationDataSources) *IntegrationSummarySources {
	out := &IntegrationSummarySources{TeslaMateConnected: s.TeslaMate, PendingCharges: s.PendingCharges}
	for _, source := range s.Sources {
		if source.LastRecord != nil && (out.LastRecord == nil || *source.LastRecord > *out.LastRecord) {
			out.LastRecord = source.LastRecord
		}
	}
	for _, sync := range s.Sync {
		if sync.LastSuccess != nil && (out.LastSync == nil || *sync.LastSuccess > *out.LastSync) {
			out.LastSync = sync.LastSuccess
		}
	}
	return out
}

// tireStats reads the wear of every tire of a vehicle with the forecasts attached.
func (h *IntegrationReadHandler) tireStats(ctx context.Context, vehicle *models.Vehicle) ([]services.TireWearStats, error) {
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
	return stats, nil
}

// ownershipOf shapes the ownership figures from a TCO already computed, so the summary does not compute it twice.
func (h *IntegrationReadHandler) ownershipOf(ctx context.Context, vehicle *models.Vehicle, sum *services.TCOSummary) (IntegrationOwnership, error) {
	var residual *models.ResidualValue
	unavailable := ""
	if h.residual != nil {
		var err error
		residual, err = h.residual.Residual(ctx, vehicle, services.ResidualOptions{KmShare: 0.5, HealthWeight: 1})
		if err != nil {
			apiErr, ok := apierror.As(err)
			if !ok {
				return IntegrationOwnership{}, err
			}
			residual, unavailable = nil, apiErr.Message
		}
	}
	return buildIntegrationOwnership(vehicle, sum, residual, unavailable), nil
}

// addSummarySections fills the optional sections asked for. pending is the account's count of unclaimed charges.
func (h *IntegrationReadHandler) addSummarySections(ctx context.Context, out *IntegrationSummaryVehicle, v *models.Vehicle, sum *services.TCOSummary, stats *services.EnergyStats, include SummaryInclude, pending int) error {
	if include.Tires {
		tireStats, err := h.tireStats(ctx, v)
		if err != nil {
			return err
		}
		out.Tires = summaryTires(v.ID, v.Currency, tireStats)
	}
	if include.Battery && stats != nil {
		if health := buildIntegrationBattery(stats.BatteryHealth); health != nil {
			out.Battery = health
		}
	}
	if include.Ownership {
		ownership, err := h.ownershipOf(ctx, v, sum)
		if err != nil {
			return err
		}
		out.Ownership = summaryOwnership(ownership)
	}
	if include.Sources {
		activity, err := h.repo.ListSourceActivity(ctx, v.ID)
		if err != nil {
			return err
		}
		states, err := h.repo.ListSyncStates(ctx, v.ID)
		if err != nil {
			return err
		}
		out.Sources = summarySources(buildIntegrationSources(v, activity, states, pending))
	}
	return nil
}

// summaryInclude reads the include parameter of the request, answering 400 for a section that does not exist.
func summaryInclude(w http.ResponseWriter, r *http.Request) (SummaryInclude, bool) {
	include, apiErr := parseSummaryInclude(r.URL.Query().Get("include"))
	if apiErr != nil {
		writeAPIError(w, http.StatusBadRequest, apiErr)
		return SummaryInclude{}, false
	}
	return include, true
}
