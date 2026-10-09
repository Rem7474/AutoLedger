package services

import (
	"sort"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// OdometerRef is a known odometer value at a given time, used to estimate the mileage of fill-ups entered without one.
type OdometerRef struct {
	Date     time.Time
	Odometer float64
}

// BuildOdometerRefs gathers the references that are not fill-ups: manual odometer readings and the
// odometer at the start of the ownership contract.
func BuildOdometerRefs(readings []models.OdometerCheckpoint, ownership *models.VehicleOwnership) []OdometerRef {
	refs := make([]OdometerRef, 0, len(readings)+1)
	if ownership != nil && ownership.StartOdometer != nil && *ownership.StartOdometer >= 0 {
		refs = append(refs, OdometerRef{Date: ownership.StartDate, Odometer: *ownership.StartOdometer})
	}
	for _, r := range readings {
		refs = append(refs, OdometerRef{Date: r.Date, Odometer: r.Odometer})
	}
	return refs
}

// EstimateOdometer interpolates the odometer at a given time between the closest references before and
// after it. It never extrapolates: outside the known range, or with no reference, it returns false.
func EstimateOdometer(at time.Time, refs []OdometerRef) (float64, bool) {
	var prev, next *OdometerRef
	for i := range refs {
		r := &refs[i]
		if !r.Date.After(at) && (prev == nil || r.Date.After(prev.Date) || (r.Date.Equal(prev.Date) && r.Odometer > prev.Odometer)) {
			prev = r
		}
		if !r.Date.Before(at) && (next == nil || r.Date.Before(next.Date) || (r.Date.Equal(next.Date) && r.Odometer < next.Odometer)) {
			next = r
		}
	}
	if prev == nil || next == nil {
		return 0, false
	}
	span := next.Date.Sub(prev.Date)
	if span <= 0 {
		return prev.Odometer, true
	}
	frac := float64(at.Sub(prev.Date)) / float64(span)
	return prev.Odometer + (next.Odometer-prev.Odometer)*frac, true
}

// AnnotatedFuelLog is a fill-up with the figures measured on the segment it closes.
// Consumption is only known for a full tank that follows another full tank, with litres
// recorded on every fill in between (the "full-to-full" method). A fill-up entered without
// mileage carries an estimated one, and the segments that rely on it are flagged as estimated.
type AnnotatedFuelLog struct {
	models.FuelLog
	OdometerEstimated *float64 `json:"odometer_estimated,omitempty"`
	SegmentKm         *float64 `json:"segment_km,omitempty"`
	SegmentEstimated  bool     `json:"segment_estimated,omitempty"`
	ConsumptionL100   *float64 `json:"consumption_l_100km,omitempty"`
	CostPerKm         *float64 `json:"cost_per_km,omitempty"`
}

// FuelStats summarizes the fill-ups of a vehicle.
type FuelStats struct {
	TotalCost        money.Cents        `json:"total_cost"`
	TotalLiters      float64            `json:"total_liters"` // Litres of the fill-ups where they are recorded
	AvgPricePerLiter float64            `json:"avg_price_per_liter"`
	ConsumptionL100  *float64           `json:"consumption_l_100km,omitempty"` // Over all measurable segments
	MeasuredKm       float64            `json:"measured_km"`
	MeasurableCount  int                `json:"measurable_segments"`
	EstimatedCount   int                `json:"estimated_segments"`    // Measurable segments relying on an estimated mileage
	UnmeasurableCnt  int                `json:"unmeasurable_segments"` // Full-to-full segments lacking litres or distance
	NoMileageCount   int                `json:"fill_ups_without_mileage"`
	FillUps          int                `json:"fill_ups"`
	Logs             []AnnotatedFuelLog `json:"logs"`
}

// ComputeFuelStats is a pure function over the fill-ups of one vehicle (any order). refs are the
// odometer references other than the fill-ups themselves (readings, contract start).
func ComputeFuelStats(logs []models.FuelLog, refs []OdometerRef) FuelStats {
	sorted := append([]models.FuelLog(nil), logs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].Date.Equal(sorted[j].Date) {
			return sorted[i].Date.Before(sorted[j].Date)
		}
		if sorted[i].Odometer == nil || sorted[j].Odometer == nil {
			return sorted[i].Odometer != nil && sorted[j].Odometer == nil
		}
		return *sorted[i].Odometer < *sorted[j].Odometer
	})

	// The fill-ups that carry a mileage are references too.
	allRefs := append([]OdometerRef(nil), refs...)
	for _, f := range sorted {
		if f.Odometer != nil {
			allRefs = append(allRefs, OdometerRef{Date: f.Date, Odometer: *f.Odometer})
		}
	}

	st := FuelStats{FillUps: len(sorted), Logs: make([]AnnotatedFuelLog, len(sorted))}
	effective := make([]*float64, len(sorted)) // Mileage of each fill-up: entered or estimated
	estimated := make([]bool, len(sorted))
	var pricedCost, pricedLiters float64
	for i, f := range sorted {
		st.TotalCost += f.Amount
		st.Logs[i] = AnnotatedFuelLog{FuelLog: f}
		if f.Liters != nil && *f.Liters > 0 {
			st.TotalLiters += *f.Liters
			pricedCost += f.Amount.Float()
			pricedLiters += *f.Liters
		}
		switch {
		case f.Odometer != nil:
			effective[i] = f.Odometer
		default:
			st.NoMileageCount++
			if v, ok := EstimateOdometer(f.Date, allRefs); ok {
				v = round1(v)
				effective[i], estimated[i] = &v, true
				st.Logs[i].OdometerEstimated = &v
			}
		}
	}
	if pricedLiters > 0 {
		st.AvgPricePerLiter = round3(pricedCost / pricedLiters)
	}

	// Full-to-full segments: the litres added at a full tank refill what was used since the previous full tank.
	anchor := -1
	var segLiters, segCost float64
	segComplete, segEstimated := true, false
	var measuredLiters float64
	for i, f := range sorted {
		if anchor >= 0 {
			segCost += f.Amount.Float()
			if f.Liters == nil || *f.Liters <= 0 {
				segComplete = false
			} else {
				segLiters += *f.Liters
			}
		}
		if !f.IsFullTank {
			continue
		}
		if anchor >= 0 {
			segEstimated = segEstimated || estimated[i] || estimated[anchor]
			var km float64
			if effective[i] != nil && effective[anchor] != nil {
				km = round1(*effective[i] - *effective[anchor])
			}
			// A segment smaller than the reporting precision cannot support a rate.
			if segComplete && km > 0 {
				l100 := round3(segLiters / km * 100)
				cpk := round3(segCost / km)
				st.Logs[i].SegmentKm = &km
				st.Logs[i].SegmentEstimated = segEstimated
				st.Logs[i].ConsumptionL100 = &l100
				st.Logs[i].CostPerKm = &cpk
				st.MeasurableCount++
				if segEstimated {
					st.EstimatedCount++
				}
				st.MeasuredKm += km
				measuredLiters += segLiters
			} else {
				st.UnmeasurableCnt++
			}
		}
		anchor, segLiters, segCost, segComplete, segEstimated = i, 0, 0, true, false
	}
	if st.MeasuredKm > 0 {
		v := round3(measuredLiters / st.MeasuredKm * 100)
		st.ConsumptionL100 = &v
	}
	st.TotalLiters = round1(st.TotalLiters)
	return st
}
