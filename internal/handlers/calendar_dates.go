package handlers

import "time"

// calendarDates gives fuel and period-filter handlers the same calendar as the reporting services.
// Date-only database fields retain parseDate's UTC default unless a location is explicitly supplied.
type calendarDates struct{ loc *time.Location }

func (d *calendarDates) SetTimezone(zone string) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc = time.UTC
	}
	d.loc = loc
}

func (d calendarDates) dateLocation() *time.Location {
	return dateLocation([]*time.Location{d.loc})
}

func dateLocation(locations []*time.Location) *time.Location {
	if len(locations) > 0 && locations[0] != nil {
		return locations[0]
	}
	return time.UTC
}
