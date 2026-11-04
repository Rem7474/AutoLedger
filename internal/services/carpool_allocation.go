package services

import (
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// AllocateCarpoolCosts splits the cost of every leg equally between the people on board during that leg
// (the driver counts as one person, a passenger counts for each seat booked). A passenger's share is the sum
// of the legs ridden between the boarding and alighting stops; the driver keeps the rest, including the
// rounding remainders. Legs, passengers and the returned shares are computed in exact cents.
func AllocateCarpoolCosts(legs []models.CarpoolLeg, passengers []models.CarpoolPassenger) (driverShare, passengersShare money.Cents) {
	for i := range passengers {
		passengers[i].CostShare = 0
	}
	for i := range legs {
		leg := &legs[i]
		leg.TotalCost = leg.Total()
		leg.PassengerSeats = 0
		for _, p := range passengers {
			if p.BoardStopIndex <= i && i < p.AlightStopIndex {
				leg.PassengerSeats += p.Seats
			}
		}

		parts := money.Split(leg.TotalCost, 1+leg.PassengerSeats)
		perPerson := parts[0]
		leg.CostPerPerson = perPerson

		var legPassengers money.Cents
		for j := range passengers {
			p := &passengers[j]
			if p.BoardStopIndex <= i && i < p.AlightStopIndex {
				share := perPerson * money.Cents(p.Seats)
				p.CostShare += share
				legPassengers += share
			}
		}
		passengersShare += legPassengers
		driverShare += leg.TotalCost - legPassengers
	}
	for i := range passengers {
		passengers[i].Balance = passengers[i].AmountPaid - passengers[i].CostShare
	}
	return driverShare, passengersShare
}

// CarpoolShareTotals sums, over several carpools, the share of the driver and the share of the passengers.
func CarpoolShareTotals(trips []models.CarpoolTripWithPassengers) (driverShare, passengersShare money.Cents) {
	for i := range trips {
		driver, passengers := AllocateCarpoolCosts(trips[i].Legs, trips[i].Passengers)
		driverShare += driver
		passengersShare += passengers
	}
	return driverShare, passengersShare
}
