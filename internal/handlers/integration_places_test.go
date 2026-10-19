package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationPlacesNamePlacesAndTheirAddressesAndKeepUnlinkedLocationsApart(t *testing.T) {
	home, unknown := "p-home", ""
	price := 0.2
	stats := &services.ChargingPlaceStats{
		Groups: []services.PlaceGroup{
			{Key: home, Name: "Maison", Category: models.PlaceHome, Sessions: 10, KwhAdded: 300, Cost: 6000, PricePerKwh: &price, FreeSessions: 1, UnknownSessions: 2, UnknownKwh: 40},
			{Key: unknown, Sessions: 3, KwhAdded: 90},
		},
		HomeSavings: services.HomeSavings{Reason: services.HomeSavingsNoAway},
	}
	locations := []models.ChargingLocation{
		{Text: "12 rue des Lilas, Lyon", Sessions: 9, KwhAdded: 250, PlaceID: &home},
		{Text: "45.7, 4.8", Sessions: 5, KwhAdded: 80, PlaceID: &home},
		{Text: "Station Autoroute A7", Sessions: 3, KwhAdded: 90.04},
	}

	out := buildIntegrationPlaces("v1", "CHF", 12, stats, locations)

	if len(out.Places) != 2 || out.Places[0].Name == nil || *out.Places[0].Name != "Maison" || out.Places[0].Category != "HOME" ||
		len(out.Places[0].Addresses) != 1 || out.Places[0].Addresses[0] != "12 rue des Lilas, Lyon" || out.Places[0].Cost.Amount != "60.00" ||
		out.Places[0].PricePerKwh == nil || out.Places[0].PricePerKwh.Amount != "0.2000" || out.Places[0].Unpriced != 2 {
		t.Fatalf("a named place with its address and without the position: %+v", out.Places[0])
	}
	if out.Places[1].Name != nil || out.Places[1].Category != "UNPLACED" || out.Places[1].Addresses == nil || out.Places[1].PricePerKwh != nil {
		t.Errorf("the charges linked to no place: %+v", out.Places[1])
	}
	if len(out.Unplaced) != 1 || out.Unplaced[0].Text != "Station Autoroute A7" || out.Unplaced[0].EnergyKwh != 90 {
		t.Errorf("a location linked to no place is offered to the owner: %+v", out.Unplaced)
	}
	if out.Home != nil || out.HomeWhy == nil || *out.HomeWhy != "no_away" {
		t.Errorf("no comparison is null with its reason, not zero: %+v %v", out.Home, out.HomeWhy)
	}
}

func TestIntegrationPlacesComparesWithTheHomePriceWhenItCan(t *testing.T) {
	home := 0.15
	stats := &services.ChargingPlaceStats{HomeSavings: services.HomeSavings{HomePricePerKwh: &home, AwaySessions: 4, AwayCost: 5000, HypotheticalCost: 3000, Savings: 2000}}
	out := buildIntegrationPlaces("v1", "CHF", 0, stats, nil)

	if out.Home == nil || out.HomeWhy != nil || out.Home.Savings.Amount != "20.00" || out.Home.HomePricePerKwh.Amount != "0.1500" || out.Places == nil || out.Unplaced == nil {
		t.Errorf("home savings: %+v", out)
	}
}
