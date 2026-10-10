package services

import (
	"sort"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

// Sources of the months a tire is expected on the car.
const (
	TireForecastMonthsLearned = "learned" // read from the past mount sessions of tires of the same season
	TireForecastMonthsDefault = "default" // typical season windows, no usable history
)

const (
	tireForecastHorizonYears = 10
	tireForecastWindowMonths = 12
	tireForecastMinWindow    = 30 * 24 * time.Hour
)

// TireForecast predicts when a tire reaches its legal wear limit.
type TireForecast struct {
	// ReplacementDate is the first day the tire is expected to be at its limit (YYYY-MM-DD); empty when it cannot be predicted.
	ReplacementDate string `json:"replacement_date,omitempty"`
	// MonthlyKm is the vehicle's average distance per month while the forecast is computed.
	MonthlyKm float64 `json:"monthly_km"`
	// MountedMonths lists the calendar months (1-12) the tire is expected on the car.
	MountedMonths []int `json:"mounted_months"`
	// MonthsSource tells whether MountedMonths was learned from history or is the default for the season.
	MonthsSource string `json:"months_source"`
}

// TireForecastInput is everything the projection needs; it holds no database access so it stays unit-testable.
type TireForecastInput struct {
	Now         time.Time
	RemainingKm float64
	Season      models.TireSeason
	// SeasonSessions are the mount sessions of every tire of this season on the vehicle, the tire's own included.
	SeasonSessions []models.TireMountSession
	Anchors        []OdometerAnchor
}

// defaultTireMonths gives the typical months a tire of a season is on the car in the northern hemisphere.
func defaultTireMonths(season models.TireSeason) [12]bool {
	var m [12]bool
	for i := range m {
		switch season {
		case models.TireSeasonWinter:
			m[i] = i >= 10 || i <= 2 // November to March
		case models.TireSeasonSummer:
			m[i] = i >= 3 && i <= 9 // April to October
		default:
			m[i] = true
		}
	}
	return m
}

// learnTireMonths reads the calendar months covered by finished mount sessions. ok is false without any.
func learnTireMonths(sessions []models.TireMountSession) (months [12]bool, ok bool) {
	for _, s := range sessions {
		if s.DismountedDate == nil || s.DismountedDate.Before(s.MountedDate) {
			continue
		}
		ok = true
		d := time.Date(s.MountedDate.Year(), s.MountedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(s.DismountedDate.Year(), s.DismountedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 12 && !d.After(end); i++ {
			months[d.Month()-1] = true
			d = d.AddDate(0, 1, 0)
		}
	}
	return months, ok
}

// averageMonthlyKm is the distance driven per month over the last year, or over the history when it is shorter.
func averageMonthlyKm(anchors []OdometerAnchor, now time.Time) float64 {
	if len(anchors) == 0 {
		return 0
	}
	first := now
	for _, a := range anchors {
		if a.Km > 0 && a.Date.Before(first) {
			first = a.Date
		}
	}
	from := now.AddDate(0, -tireForecastWindowMonths, 0)
	if from.Before(first) {
		from = first
	}
	if now.Sub(from) < tireForecastMinWindow {
		return 0
	}
	end, _, okEnd := EstimateOdometerAt(anchors, now)
	start, _, okStart := EstimateOdometerAt(anchors, from)
	if !okEnd || !okStart || end <= start {
		return 0
	}
	months := now.Sub(from).Hours() / 24 / (365.0 / 12)
	return (end - start) / months
}

// ForecastTireReplacement spreads the remaining distance of a tire over the months it is on the car at the average
// monthly mileage of the vehicle. Returns nil when the vehicle has no usable mileage history.
func ForecastTireReplacement(in TireForecastInput) *TireForecast {
	monthly := averageMonthlyKm(in.Anchors, in.Now)
	if monthly <= 0 {
		return nil
	}
	months, source := defaultTireMonths(in.Season), TireForecastMonthsDefault
	if in.Season != models.TireSeasonAllSeason {
		if learned, ok := learnTireMonths(in.SeasonSessions); ok {
			months, source = learned, TireForecastMonthsLearned
		}
	}
	f := &TireForecast{MonthlyKm: monthly, MonthsSource: source}
	for i, on := range months {
		if on {
			f.MountedMonths = append(f.MountedMonths, i+1)
		}
	}
	sort.Ints(f.MountedMonths)

	remaining := in.RemainingKm
	if remaining <= 0 {
		f.ReplacementDate = in.Now.Format("2006-01-02")
		return f
	}
	perDay := monthly * 12 / 365
	day := time.Date(in.Now.Year(), in.Now.Month(), in.Now.Day(), 0, 0, 0, 0, time.UTC)
	limit := day.AddDate(tireForecastHorizonYears, 0, 0)
	for ; day.Before(limit); day = day.AddDate(0, 0, 1) {
		if !months[day.Month()-1] {
			continue
		}
		remaining -= perDay
		if remaining <= 0 {
			f.ReplacementDate = day.Format("2006-01-02")
			return f
		}
	}
	return f
}
