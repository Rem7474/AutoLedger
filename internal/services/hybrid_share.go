package services

import (
	"math"
	"sort"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

// Thresholds of the suggested electric share of a plug-in hybrid or range extender.
const (
	hybridMinSegments     = 3     // closed full-to-full periods needed
	hybridMinKm           = 500.0 // cumulated distance of those periods
	hybridWindowMonths    = 12
	hybridMinSegmentKm    = 20.0 // a shorter period cannot support a ratio
	hybridIncoherentRatio = 1.15 // electric distance beyond this multiple of the measured one: discarded
	hybridMinSocGap       = 30   // battery percentage points a charge must span to give the capacity
	hybridMinConditioning = 0.05 // det/(Σe²Σl²) below this, energy and litres are too correlated to be told apart
)

// Levels of confidence of a suggestion.
const (
	HybridLevelInsufficient = "insufficient"
	HybridLevelLow          = "low"
	HybridLevelGood         = "good"
)

// Methods used to split the distance.
const (
	HybridMethodReference  = "reference"  // the entered reference consumption converts kWh into km
	HybridMethodRegression = "regression" // km = a·kWh + b·L fitted on the periods
)

// HybridShareInput is everything the suggestion reads, with no database access.
type HybridShareInput struct {
	Fuel    []models.FuelLog
	Charges []models.ChargeLog
	Refs    []OdometerRef // odometer references other than the fill-ups (readings, contract start)
	// ReferenceKwh100 is the electric consumption entered on the vehicle, read on the plug side (kWh/100 km).
	ReferenceKwh100 *float64
	Now             time.Time
}

// HybridShareQuality says how far the suggestion can be trusted, and why not when it cannot.
type HybridShareQuality struct {
	SegmentsUsed         int     `json:"segments_used"`
	SegmentsDiscarded    int     `json:"segments_discarded"`
	DiscardedNoOdometer  int     `json:"discarded_no_odometer"`
	DiscardedNoLiters    int     `json:"discarded_no_liters"`
	DiscardedTooShort    int     `json:"discarded_too_short"`
	DiscardedIncoherent  int     `json:"discarded_incoherent"`
	KmCovered            float64 `json:"km_covered"`
	KmTotal              float64 `json:"km_total"`
	CoherenceGap         float64 `json:"coherence_gap"` // median relative gap between the measured distance and the modelled one
	SocCoverage          float64 `json:"soc_coverage"`  // share of the periods corrected with battery percentages
	LessPreciseEnergy    int     `json:"less_precise_energy_charges"`
	Level                string  `json:"level"`
	InsufficientBecause  string  `json:"insufficient_because,omitempty"`
	WindowMonths         int     `json:"window_months,omitempty"` // 0: the whole history
	CapacityFromPercents bool    `json:"capacity_from_percentages"`
}

// HybridShareResult is the suggestion: never applied by itself, only proposed.
type HybridShareResult struct {
	Suggested          *float64           `json:"suggested_electric_share,omitempty"` // 0 to 1
	Method             string             `json:"method,omitempty"`
	ElectricKwh100km   *float64           `json:"electric_kwh_100km,omitempty"`     // plug side
	FuelL100kmFuelPart *float64           `json:"fuel_l_100km_fuel_part,omitempty"` // on the km driven on fuel
	BatteryKwh         *float64           `json:"battery_usable_kwh,omitempty"`     // deduced from charges with percentages
	Quality            HybridShareQuality `json:"quality"`
}

type hybridSegment struct {
	end      time.Time
	startOdo float64
	endOdo   float64
	km       float64
	liters   float64
	kwh      float64 // plug-side energy of the period
	elecKm   float64 // electric distance, once the method is known
	reason   string  // "" when usable
	socUsed  bool
	lessPrec int
}

// ComputeHybridShare proposes the share of the distance driven on electricity, from the fill-ups and charges
// the owner typed with their odometer. A period is a full-to-full segment of the fuel, the same as the fuel
// consumption. It is pure: the caller decides whether to store what it returns.
func ComputeHybridShare(in HybridShareInput) HybridShareResult {
	stats := ComputeFuelStats(in.Fuel, in.Refs)
	capacity, fromPercents := hybridBatteryKwh(in.Charges)
	var segs []hybridSegment

	anchor := -1
	var segLiters float64
	litersOK := true
	odo := func(l models.FuelLog, est *float64) *float64 {
		if l.Odometer != nil {
			return l.Odometer
		}
		return est
	}
	for i, a := range stats.Logs {
		if anchor >= 0 {
			if a.Liters == nil || *a.Liters <= 0 {
				litersOK = false
			} else {
				segLiters += *a.Liters
			}
		}
		if !a.IsFullTank {
			continue
		}
		if anchor >= 0 {
			start, end := stats.Logs[anchor], a
			so, eo := odo(start.FuelLog, start.OdometerEstimated), odo(end.FuelLog, end.OdometerEstimated)
			seg := hybridSegment{end: end.Date, liters: segLiters}
			switch {
			case so == nil || eo == nil:
				seg.reason = "no_odometer"
			case !litersOK:
				seg.reason = "no_liters"
			default:
				seg.startOdo, seg.endOdo, seg.km = *so, *eo, *eo-*so
				if seg.km < hybridMinSegmentKm {
					seg.reason = "too_short"
				} else {
					seg.kwh, seg.lessPrec = hybridSegmentEnergy(in.Charges, start.Date, end.Date, *so, *eo)
				}
			}
			segs = append(segs, seg)
		}
		anchor, segLiters, litersOK = i, 0, true
	}
	return hybridSuggest(in, segs, capacity, fromPercents)
}

// hybridSegmentEnergy sums the plug-side energy of the charges that belong to a period: by odometer when the charge
// has one, by date otherwise. A charge without a plug-side figure falls back to kwh_added: exact for a manual charge, the battery side (and so counted as less precise) for a synchronised one.
func hybridSegmentEnergy(charges []models.ChargeLog, from, to time.Time, fromOdo, toOdo float64) (kwh float64, lessPrecise int) {
	for _, c := range charges {
		in := false
		if c.Odometer != nil {
			in = *c.Odometer > fromOdo && *c.Odometer <= toOdo
		} else {
			in = c.Date.After(from) && !c.Date.After(to)
		}
		if !in {
			continue
		}
		if c.KwhUsed != nil && *c.KwhUsed > 0 {
			kwh += *c.KwhUsed
		} else {
			// A charge typed by hand carries the figure the owner read, which is the plug side. A synchronised one
			// without a plug-side figure only has the battery side.
			kwh += c.KwhAdded
			if !c.IsManual {
				lessPrecise++
			}
		}
	}
	return kwh, lessPrecise
}

// hybridBatteryKwh deduces the usable capacity: the median of kWh added over the percentage points a charge spans,
// over the charges that span at least hybridMinSocGap points. It is never asked of the owner.
func hybridBatteryKwh(charges []models.ChargeLog) (float64, bool) {
	var caps []float64
	for _, c := range charges {
		if c.StartBatteryLevel == nil || c.EndBatteryLevel == nil || c.KwhAdded <= 0 {
			continue
		}
		if gap := *c.EndBatteryLevel - *c.StartBatteryLevel; gap >= hybridMinSocGap {
			caps = append(caps, c.KwhAdded/(float64(gap)/100))
		}
	}
	if len(caps) == 0 {
		return 0, false
	}
	return median(caps), true
}

func hybridSuggest(in HybridShareInput, segs []hybridSegment, capacity float64, fromPercents bool) HybridShareResult {
	res := HybridShareResult{}
	q := &res.Quality
	q.CapacityFromPercents = fromPercents
	if fromPercents {
		c := round1(capacity)
		res.BatteryKwh = &c
	}

	// Window: the closed periods of the last months, widened to the whole history when they are too few.
	usableIn := func(since time.Time) int {
		n := 0
		for _, s := range segs {
			if s.reason == "" && !s.end.Before(since) {
				n++
			}
		}
		return n
	}
	since := models.AddCalendarMonths(in.Now, -hybridWindowMonths)
	if usableIn(since) >= hybridMinSegments {
		q.WindowMonths = hybridWindowMonths
	} else {
		since = time.Time{}
	}

	var used []*hybridSegment
	for i := range segs {
		s := &segs[i]
		if s.end.Before(since) {
			continue
		}
		q.KmTotal += math.Max(s.km, 0)
		switch s.reason {
		case "no_odometer":
			q.DiscardedNoOdometer++
		case "no_liters":
			q.DiscardedNoLiters++
		case "too_short":
			q.DiscardedTooShort++
		default:
			used = append(used, s)
		}
	}

	reference := in.ReferenceKwh100 != nil && *in.ReferenceKwh100 > 0
	if reference {
		used = hybridWithReference(in, used, capacity, fromPercents, q)
	}
	q.SegmentsDiscarded = q.DiscardedNoOdometer + q.DiscardedNoLiters + q.DiscardedTooShort + q.DiscardedIncoherent
	q.SegmentsUsed = len(used)
	for _, s := range used {
		q.KmCovered += s.km
	}
	q.KmCovered = round1(q.KmCovered)
	q.KmTotal = round1(q.KmTotal)

	if len(used) < hybridMinSegments || q.KmCovered < hybridMinKm {
		q.Level, q.InsufficientBecause = HybridLevelInsufficient, "not_enough_data"
		return res
	}
	if reference {
		res.Method = HybridMethodReference
	} else if !hybridRegression(used, &res) {
		res.Method = ""
		q.Level, q.InsufficientBecause = HybridLevelInsufficient, "energy_and_fuel_not_separable"
		return res
	} else {
		res.Method = HybridMethodRegression
	}

	var elec, km, fuelKm, fuelL float64
	for _, s := range used {
		elec += s.elecKm
		km += s.km
		fuelKm += s.km - s.elecKm
		fuelL += s.liters
		if s.socUsed {
			q.SocCoverage++
		}
		q.LessPreciseEnergy += s.lessPrec
	}
	q.SocCoverage = round3(q.SocCoverage / float64(len(used)))
	share := round3(clamp01(elec / km))
	res.Suggested = &share
	if reference {
		v := round1(*in.ReferenceKwh100)
		res.ElectricKwh100km = &v
	}
	if fuelKm > hybridMinSegmentKm && fuelL > 0 {
		v := round3(fuelL / fuelKm * 100)
		res.FuelL100kmFuelPart = &v
	}
	q.CoherenceGap = round3(hybridCoherenceGap(used, res))

	weak := q.CoherenceGap > 0.25 || (q.SocCoverage == 0 && (len(used) < 2*hybridMinSegments || 2*q.LessPreciseEnergy > len(used)))
	if weak {
		q.Level = HybridLevelLow
	} else {
		q.Level = HybridLevelGood
	}
	return res
}

// hybridRegression fits km = a·kWh + b·L (both non-negative) on the periods and sets their electric distance.
// It refuses a series where energy and litres move together, which cannot be told apart.
func hybridRegression(used []*hybridSegment, res *HybridShareResult) bool {
	var see, sll, sel, sed, sld float64
	for _, s := range used {
		see += s.kwh * s.kwh
		sll += s.liters * s.liters
		sel += s.kwh * s.liters
		sed += s.kwh * s.km
		sld += s.liters * s.km
	}
	if see == 0 || sll == 0 {
		return false
	}
	det := see*sll - sel*sel
	if det/(see*sll) < hybridMinConditioning {
		return false
	}
	a, b := (sed*sll-sld*sel)/det, (sld*see-sed*sel)/det
	if a < 0 {
		a, b = 0, sld/sll
	} else if b < 0 {
		a, b = sed/see, 0
	}
	for _, s := range used {
		s.elecKm = math.Min(a*s.kwh, s.km)
	}
	if a > 0 {
		v := round1(100 / a)
		res.ElectricKwh100km = &v
	}
	return true
}

// hybridCoherenceGap is the median relative gap between the measured distance of a period and what the electric
// energy and the litres of the same period would cover, with the fuel consumption measured on the whole series.
func hybridCoherenceGap(used []*hybridSegment, res HybridShareResult) float64 {
	var fuelKm, fuelL float64
	for _, s := range used {
		fuelKm += s.km - s.elecKm
		fuelL += s.liters
	}
	if fuelKm <= 0 || fuelL <= 0 {
		return 0
	}
	kmPerL := fuelKm / fuelL
	gaps := make([]float64, 0, len(used))
	for _, s := range used {
		gaps = append(gaps, math.Abs(s.km-s.elecKm-s.liters*kmPerL)/s.km)
	}
	return median(gaps)
}

// hybridWithReference converts each period's energy into electric kilometres with the entered consumption. Where the
// charges carry battery percentages, the energy actually used between the first and the last charge of the period
// replaces the plug-side sum, which ignores what was left in the battery at the borders.
func hybridWithReference(in HybridShareInput, used []*hybridSegment, capacity float64, fromPercents bool, q *HybridShareQuality) []*hybridSegment {
	perKm := *in.ReferenceKwh100 / 100
	kept := used[:0:0]
	for _, s := range used {
		km, kwh := s.km, s.kwh
		if fromPercents {
			if wKm, wKwh, ok := hybridSocWindow(in.Charges, s, capacity); ok {
				km, kwh, s.socUsed = wKm, wKwh, true
			}
		}
		elec := kwh / perKm
		if elec > km*hybridIncoherentRatio {
			q.DiscardedIncoherent++
			continue
		}
		// The window's ratio applies to the whole period.
		s.elecKm = clamp01(elec/km) * s.km
		kept = append(kept, s)
	}
	return kept
}

// hybridSocWindow measures the energy drawn between the first and the last charge of a period that carry both
// battery percentages and an odometer: what the charges in between add, plus what the battery lost between the end
// of the first charge and the start of the last one.
func hybridSocWindow(charges []models.ChargeLog, s *hybridSegment, capacity float64) (km, kwh float64, ok bool) {
	var inside []models.ChargeLog
	for _, c := range charges {
		if c.Odometer == nil || c.StartBatteryLevel == nil || c.EndBatteryLevel == nil {
			continue
		}
		if *c.Odometer > s.startOdo && *c.Odometer <= s.endOdo {
			inside = append(inside, c)
		}
	}
	if len(inside) < 2 {
		return 0, 0, false
	}
	sort.Slice(inside, func(i, j int) bool { return *inside[i].Odometer < *inside[j].Odometer })
	first, last := inside[0], inside[len(inside)-1]
	km = *last.Odometer - *first.Odometer
	if km < hybridMinSegmentKm {
		return 0, 0, false
	}
	for _, c := range inside[1 : len(inside)-1] {
		kwh += c.KwhAdded
	}
	kwh += capacity * float64(*first.EndBatteryLevel-*last.StartBatteryLevel) / 100
	if kwh < 0 {
		return 0, 0, false
	}
	return km, kwh, true
}
