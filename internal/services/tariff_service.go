package services

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// TariffService computes charging costs for time-of-use (HP/HC) and public charging schemes.
type TariffService struct {
	loc *time.Location
}

func NewTariffService() *TariffService {
	return &TariffService{loc: time.UTC}
}

// NewTariffServiceIn reads the time of day of a session in the given IANA timezone (APP_TIMEZONE): off-peak hours
// are wall-clock hours.
func NewTariffServiceIn(timezone string) *TariffService {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("unknown timezone for tariffs, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}
	return &TariffService{loc: loc}
}

// DayOf is the calendar day (YYYY-MM-DD) of a moment in the service timezone, used to pick the version of a tariff.
func (s *TariffService) DayOf(t time.Time) string {
	return t.In(s.loc).Format("2006-01-02")
}

// parseTimeToMinutesOfDay converts "HH:MM" into minutes from 00:00 (0..1439).
func parseTimeToMinutesOfDay(timeStr string) (int, error) {
	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format %q, expected HH:MM", timeStr)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour %q", parts[0])
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid minute %q", parts[1])
	}
	return h*60 + m, nil
}

// effectiveGrid reads any plan as bands, rules and a default band: a flat plan is one band, a peak / off-peak plan
// two bands with a rule per off-peak window. ok is false when the plan carries no usable price.
func effectiveGrid(plan *models.TariffPlan) (bands map[string]money.Rate, rules []models.TariffRule, defaultBand string, ok bool) {
	bands = map[string]money.Rate{}
	if plan.PlanType == models.TariffTypeBands {
		for _, b := range plan.Bands {
			bands[b.Name] = b.RateCents
		}
		defaultBand = plan.DefaultBand
		if _, found := bands[defaultBand]; !found && len(plan.Bands) > 0 {
			defaultBand = plan.Bands[0].Name
		}
		return bands, plan.Rules, defaultBand, len(plan.Bands) > 0
	}

	if plan.PlanType == models.TariffTypeFlat || len(plan.TimeWindows) == 0 {
		switch {
		case plan.FlatRateCents != nil:
			bands["FLAT"] = *plan.FlatRateCents
		case plan.OffpeakRateCents != nil:
			bands["FLAT"] = *plan.OffpeakRateCents
		default:
			return bands, nil, "", false
		}
		return bands, nil, "FLAT", true
	}

	bands[models.TimeWindowPeak] = 0
	bands[models.TimeWindowOffPeak] = 0
	if plan.PeakRateCents != nil {
		bands[models.TimeWindowPeak] = *plan.PeakRateCents
	}
	if plan.OffpeakRateCents != nil {
		bands[models.TimeWindowOffPeak] = *plan.OffpeakRateCents
	}
	for _, w := range plan.TimeWindows {
		band := models.TimeWindowPeak
		if strings.ToUpper(strings.TrimSpace(w.Kind)) == models.TimeWindowOffPeak {
			band = models.TimeWindowOffPeak
		}
		rules = append(rules, models.TariffRule{Start: w.Start, End: w.End, Band: band})
	}
	return bands, rules, models.TimeWindowPeak, true
}

type parsedRule struct {
	days     [7]bool
	startMin int
	endMin   int
	band     string
}

func parseRules(rules []models.TariffRule) []parsedRule {
	parsed := make([]parsedRule, 0, len(rules))
	for _, r := range rules {
		sMin, err1 := parseTimeToMinutesOfDay(r.Start)
		eMin, err2 := parseTimeToMinutesOfDay(r.End)
		if strings.TrimSpace(r.End) == "24:00" {
			eMin, err2 = 1440, nil
		}
		if err1 != nil || err2 != nil {
			continue
		}
		pr := parsedRule{startMin: sMin, endMin: eMin, band: r.Band}
		if len(r.Days) == 0 {
			for i := range pr.days {
				pr.days[i] = true
			}
		}
		for _, d := range r.Days {
			if d >= 0 && d <= 6 {
				pr.days[d] = true
			}
		}
		parsed = append(parsed, pr)
	}
	return parsed
}

// matches tells whether the rule covers the given weekday and minute of the day.
func (r parsedRule) matches(weekday time.Weekday, mod int) bool {
	if r.startMin == r.endMin {
		return r.days[weekday]
	}
	if r.startMin < r.endMin {
		return r.days[weekday] && mod >= r.startMin && mod < r.endMin
	}
	// Crosses midnight: the evening part belongs to the day it starts on, the morning part to the day before.
	previous := (int(weekday) + 6) % 7
	return (r.days[weekday] && mod >= r.startMin) || (r.days[previous] && mod < r.endMin)
}

// BandShare is the part of a session priced in one band of a plan.
type BandShare struct {
	Band    string
	Minutes int
	Kwh     float64
	Rate    money.Rate
}

// SessionPricing is the cost of a session and how it splits between the bands of the plan, the largest share first.
type SessionPricing struct {
	Cost  money.Cents
	Bands []BandShare
}

// CalculateSessionCost splits the energy of a session between the bands in force while it ran, in proportion to
// the time spent in each, and returns the cost in cents. The clock is read in the service timezone, so a window
// follows the wall clock across daylight-saving changes. A plan whose validity range excludes the day of the
// session prices nothing.
func (s *TariffService) CalculateSessionCost(plan *models.TariffPlan, startTime, endTime time.Time, kwh float64) (money.Cents, error) {
	pricing, err := s.PriceSession(plan, startTime, endTime, kwh)
	return pricing.Cost, err
}

// maxSampledSessionMinutes is how much of a session is walked to find its bands: a year, which is far beyond any
// charge and keeps the walk at about 105,000 steps whatever the period asked for.
const maxSampledSessionMinutes = 366 * 24 * 60

// sessionSampling says how a session of totalMinutes is walked to find its bands: every minute up to a day, every 5
// minutes beyond, and only over the first maxSampledSessionMinutes. The shares are scaled to the whole duration by the
// caller, and the bound matters because the period comes straight from the request.
func sessionSampling(totalMinutes int) (step, sampled int) {
	step = 1
	if totalMinutes > 1440 {
		step = 5
	}
	sampled = totalMinutes
	if sampled > maxSampledSessionMinutes {
		sampled = maxSampledSessionMinutes
	}
	return step, sampled
}

// PriceSession is CalculateSessionCost with the bands the session was priced in.
func (s *TariffService) PriceSession(plan *models.TariffPlan, startTime, endTime time.Time, kwh float64) (SessionPricing, error) {
	if plan == nil || kwh <= 0 {
		return SessionPricing{}, nil
	}
	startTime = startTime.In(s.loc)
	if !plan.AppliesOn(startTime.Format("2006-01-02")) {
		return SessionPricing{}, nil
	}

	bands, rules, defaultBand, ok := effectiveGrid(plan)
	if !ok {
		return SessionPricing{}, nil
	}
	if !endTime.After(startTime) {
		endTime = startTime.Add(1 * time.Minute)
	}
	totalMinutes := int(math.Max(1, math.Round(endTime.Sub(startTime).Minutes())))
	if len(rules) == 0 {
		return SessionPricing{
			Cost:  bands[defaultBand].Cost(kwh),
			Bands: []BandShare{{Band: defaultBand, Minutes: totalMinutes, Kwh: kwh, Rate: bands[defaultBand]}},
		}, nil
	}
	parsed := parseRules(rules)

	step, sampled := sessionSampling(totalMinutes)

	minutesByBand := map[string]int{}
	evaluated := 0
	for m := 0; m < sampled; m += step {
		curr := startTime.Add(time.Duration(m) * time.Minute)
		mod := curr.Hour()*60 + curr.Minute()
		band := defaultBand
		for _, r := range parsed {
			if r.matches(curr.Weekday(), mod) {
				band = r.band
				break
			}
		}
		minutesByBand[band] += step
		evaluated += step
	}

	shares := make([]BandShare, 0, len(minutesByBand))
	for band, minutes := range minutesByBand {
		fraction := float64(minutes) / float64(evaluated)
		shares = append(shares, BandShare{Band: band, Minutes: int(math.Round(fraction * float64(totalMinutes))), Kwh: kwh * fraction, Rate: bands[band]})
	}
	sort.Slice(shares, func(i, j int) bool {
		if shares[i].Kwh != shares[j].Kwh {
			return shares[i].Kwh > shares[j].Kwh
		}
		return shares[i].Band < shares[j].Band
	})
	cost := 0.0
	for _, share := range shares {
		cost += share.Kwh * share.Rate.Float()
	}
	return SessionPricing{Cost: money.FromFloat(cost), Bands: shares}, nil
}

// CalculateSessionCostIfApplicable distinguishes a known free session from an unknown cost.
// An absent, unusable or out-of-date plan leaves the cost nil; an applicable zero rate returns a pointer to zero.
func (s *TariffService) CalculateSessionCostIfApplicable(plan *models.TariffPlan, startTime, endTime time.Time, kwh float64) (*money.Cents, error) {
	if plan == nil || kwh <= 0 || !plan.AppliesOn(s.DayOf(startTime)) {
		return nil, nil
	}
	if plan.PlanType == models.TariffTypeTimeOfUse && len(plan.TimeWindows) > 0 && (plan.PeakRateCents == nil || plan.OffpeakRateCents == nil) {
		return nil, nil
	}
	if _, _, _, ok := effectiveGrid(plan); !ok {
		return nil, nil
	}
	cost, err := s.CalculateSessionCost(plan, startTime, endTime, kwh)
	if err != nil {
		return nil, err
	}
	return &cost, nil
}

// CalculatePublicCharging computes decomposed public charging fees (connection + energy + time + idle).
func (s *TariffService) CalculatePublicCharging(req models.PublicChargingCalculationRequest) models.PublicChargingBreakdown {
	energyCost := req.PricePerKwh.Cost(req.Kwh)
	durationCost := money.Cents(int64(req.ChargingMinutes) * int64(req.PricePerMinute))

	totalPlugged := req.TotalPluggedMinutes
	if totalPlugged < req.ChargingMinutes {
		totalPlugged = req.ChargingMinutes
	}
	idleMinutes := totalPlugged - req.ChargingMinutes
	billableIdle := 0
	if idleMinutes > req.IdleGraceMinutes {
		billableIdle = idleMinutes - req.IdleGraceMinutes
	}
	idleCost := money.Cents(int64(billableIdle) * int64(req.IdleFeePerMinute))

	totalCost := req.ConnectionFee + energyCost + durationCost + idleCost

	return models.PublicChargingBreakdown{
		ConnectionCost: req.ConnectionFee,
		EnergyCost:     energyCost,
		DurationCost:   durationCost,
		IdleMinutes:    idleMinutes,
		IdleCost:       idleCost,
		TotalCost:      totalCost,
	}
}

// Grid reads any plan as named bands, rules and a default band, the way sessions are priced. A band plan keeps the
// order of its bands; the others list theirs by name. ok is false when the plan carries no usable price.
func (s *TariffService) Grid(plan *models.TariffPlan) (bands []models.TariffBand, rules []models.TariffRule, defaultBand string, ok bool) {
	if plan == nil {
		return nil, nil, "", false
	}
	rates, rules, defaultBand, ok := effectiveGrid(plan)
	if !ok {
		return nil, nil, "", false
	}
	if plan.PlanType == models.TariffTypeBands {
		return plan.Bands, rules, defaultBand, true
	}
	names := make([]string, 0, len(rates))
	for name := range rates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		bands = append(bands, models.TariffBand{Name: name, RateCents: rates[name]})
	}
	return bands, rules, defaultBand, true
}
