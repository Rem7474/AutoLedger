package services

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestAllocateCarpoolCostsWithBoardingAndAlighting(t *testing.T) {
	// Annecy → Chambéry (30 €) → Grenoble (30 €) → Valence (60 €)
	legs := []models.CarpoolLeg{
		{TollsCost: 1000, EnergyCost: 2000},
		{EnergyCost: 3000},
		{TollsCost: 2000, EnergyCost: 4000},
	}
	passengers := []models.CarpoolPassenger{
		{PassengerName: "Anna", Seats: 1, BoardStopIndex: 0, AlightStopIndex: 3, AmountPaid: 4000},  // whole trip
		{PassengerName: "Bruno", Seats: 1, BoardStopIndex: 1, AlightStopIndex: 3, AmountPaid: 2500}, // boards at Chambéry
		{PassengerName: "Chloé", Seats: 2, BoardStopIndex: 0, AlightStopIndex: 1, AmountPaid: 1000}, // 2 seats, leaves at Chambéry
	}

	driver, shared := AllocateCarpoolCosts(legs, passengers)

	// Leg 0: 30 € / (driver + Anna + 2 seats Chloé) = 7.50 € per person
	// Leg 1: 30 € / (driver + Anna + Bruno) = 10 €
	// Leg 2: 60 € / 3 = 20 €
	if legs[0].PassengerSeats != 3 || legs[0].CostPerPerson != 750 || legs[1].CostPerPerson != 1000 || legs[2].CostPerPerson != 2000 {
		t.Fatalf("unexpected leg split: %+v", legs)
	}
	if passengers[0].CostShare != 3750 || passengers[1].CostShare != 3000 || passengers[2].CostShare != 1500 {
		t.Fatalf("unexpected passenger shares: %d %d %d", passengers[0].CostShare, passengers[1].CostShare, passengers[2].CostShare)
	}
	if passengers[1].Balance != -500 || passengers[2].Balance != -500 || passengers[0].Balance != 250 {
		t.Fatalf("unexpected balances: %d %d %d", passengers[0].Balance, passengers[1].Balance, passengers[2].Balance)
	}
	if driver != 3750 || shared != 8250 || driver+shared != 12000 {
		t.Fatalf("driver share %d + passengers %d must equal the 120 € total", driver, shared)
	}
}

func TestAllocateCarpoolCostsRoundingGoesToDriver(t *testing.T) {
	legs := []models.CarpoolLeg{{EnergyCost: 1000}} // 10 € for 3 people
	passengers := []models.CarpoolPassenger{
		{Seats: 1, BoardStopIndex: 0, AlightStopIndex: 1},
		{Seats: 1, BoardStopIndex: 0, AlightStopIndex: 1},
	}
	driver, shared := AllocateCarpoolCosts(legs, passengers)
	if passengers[0].CostShare != 333 || passengers[1].CostShare != 333 || driver != 334 || shared != 666 {
		t.Fatalf("unexpected rounding: passengers %d/%d, driver %d", passengers[0].CostShare, passengers[1].CostShare, driver)
	}
}

func TestAllocateCarpoolCostsEmptyLegIsDriverOnly(t *testing.T) {
	legs := []models.CarpoolLeg{{EnergyCost: 500}, {EnergyCost: 900}}
	passengers := []models.CarpoolPassenger{{Seats: 1, BoardStopIndex: 1, AlightStopIndex: 2}}
	driver, _ := AllocateCarpoolCosts(legs, passengers)
	if legs[0].PassengerSeats != 0 || driver != 500+450 || passengers[0].CostShare != 450 {
		t.Fatalf("driver alone on the first leg: driver %d, passenger %d", driver, passengers[0].CostShare)
	}
}
