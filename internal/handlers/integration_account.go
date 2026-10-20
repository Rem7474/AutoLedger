package handlers

import (
	"context"
	"time"
)

// IntegrationAccountVehicle is a vehicle of the account with the currency its amounts are in.
type IntegrationAccountVehicle struct {
	ID       string `json:"vehicle_id" jsonschema:"the vehicle's id"`
	Name     string `json:"name" jsonschema:"the vehicle's name, user text"`
	Currency string `json:"currency" jsonschema:"ISO 4217 currency of every amount of this vehicle"`
}

// IntegrationAccount is how the account reads its figures: the units and the calendar the app shows them in.
type IntegrationAccount struct {
	DistanceUnit    string                      `json:"distance_unit" jsonschema:"km or mi: how the owner reads distances. Every distance the tools return is in kilometres whatever this says: convert only to answer in the owner's unit"`
	VolumeUnit      string                      `json:"volume_unit" jsonschema:"l, gal_us or gal_uk: how the owner reads fuel volumes. The tools return litres"`
	Language        string                      `json:"language" jsonschema:"en or fr: the owner's language"`
	ReportingZone   string                      `json:"reporting_timezone" jsonschema:"IANA timezone whose calendar months bound every monthly figure"`
	MonthInProgress string                      `json:"month_in_progress" jsonschema:"the current calendar month in that timezone, YYYY-MM; its figures are not final"`
	Vehicles        []IntegrationAccountVehicle `json:"vehicles" jsonschema:"the vehicles of the account with their currency: vehicles of different currencies must not be added together"`
}

// Account describes the units, the calendar and the currencies the account reads its figures in.
func (h *IntegrationReadHandler) Account(ctx context.Context, userID string) (*IntegrationAccount, error) {
	user, err := h.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	vehicles, err := h.Vehicles(ctx, userID)
	if err != nil {
		return nil, err
	}
	loc := h.tcoService.Location()
	out := &IntegrationAccount{
		DistanceUnit: user.DistanceUnit, VolumeUnit: user.VolumeUnit, Language: user.Language,
		ReportingZone: loc.String(), MonthInProgress: time.Now().In(loc).Format("2006-01"), Vehicles: []IntegrationAccountVehicle{},
	}
	for _, v := range vehicles {
		out.Vehicles = append(out.Vehicles, IntegrationAccountVehicle{ID: v.ID, Name: v.Name, Currency: v.Currency})
	}
	return out, nil
}
