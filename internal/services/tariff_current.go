package services

import (
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// tariffChangeHorizon is how far ahead NextPriceChange looks. Rules repeat weekly, so a week and a day cover every
// change a plan can have.
const tariffChangeHorizon = 8 * 24 * time.Hour

// PlanPrice is the band in force at a moment and what it charges per kWh.
type PlanPrice struct {
	Band string
	Rate money.Rate
}

// planUsable tells whether the plan carries a complete price, the way a recorded session is priced.
func planUsable(plan *models.TariffPlan) bool {
	if plan == nil {
		return false
	}
	if plan.PlanType == models.TariffTypeTimeOfUse && len(plan.TimeWindows) > 0 && (plan.PeakRateCents == nil || plan.OffpeakRateCents == nil) {
		return false
	}
	_, _, _, ok := effectiveGrid(plan)
	return ok
}

// priceResolver reads the band of a plan at any moment; the rules are parsed once for a scan.
type priceResolver struct {
	s           *TariffService
	plan        *models.TariffPlan
	bands       map[string]money.Rate
	rules       []parsedRule
	defaultBand string
}

func (s *TariffService) resolver(plan *models.TariffPlan) (priceResolver, bool) {
	if !planUsable(plan) {
		return priceResolver{}, false
	}
	bands, rules, defaultBand, _ := effectiveGrid(plan)
	return priceResolver{s: s, plan: plan, bands: bands, rules: parseRules(rules), defaultBand: defaultBand}, true
}

// at is the price at t, false when the plan does not apply on that day.
func (r priceResolver) at(t time.Time) (PlanPrice, bool) {
	t = t.In(r.s.loc)
	if !r.plan.AppliesOn(t.Format("2006-01-02")) {
		return PlanPrice{}, false
	}
	mod := t.Hour()*60 + t.Minute()
	band := r.defaultBand
	for _, rule := range r.rules {
		if rule.matches(t.Weekday(), mod) {
			band = rule.band
			break
		}
	}
	return PlanPrice{Band: band, Rate: r.bands[band]}, true
}

// PriceAt is the band of the plan in force at t and its price per kWh, read on the wall clock of the service
// timezone. ok is false when the plan is missing, carries no complete price, or does not apply on that day.
func (s *TariffService) PriceAt(plan *models.TariffPlan, t time.Time) (PlanPrice, bool) {
	r, ok := s.resolver(plan)
	if !ok {
		return PlanPrice{}, false
	}
	return r.at(t)
}

// NextPriceChange is the first minute after t at which the price per kWh differs from the one at t, with the price
// from then on. It looks a week ahead and stops where the plan stops applying; ok is false when the price does not
// change within that range (a flat plan) or the plan has no price at t.
func (s *TariffService) NextPriceChange(plan *models.TariffPlan, t time.Time) (time.Time, PlanPrice, bool) {
	r, ok := s.resolver(plan)
	if !ok {
		return time.Time{}, PlanPrice{}, false
	}
	now, ok := r.at(t)
	if !ok {
		return time.Time{}, PlanPrice{}, false
	}
	if len(r.rules) == 0 {
		return time.Time{}, PlanPrice{}, false
	}
	start := t.Truncate(time.Minute)
	for at := start.Add(time.Minute); at.Sub(start) <= tariffChangeHorizon; at = at.Add(time.Minute) {
		price, applies := r.at(at)
		if !applies {
			return time.Time{}, PlanPrice{}, false
		}
		if price.Rate != now.Rate {
			return at, price, true
		}
	}
	return time.Time{}, PlanPrice{}, false
}
