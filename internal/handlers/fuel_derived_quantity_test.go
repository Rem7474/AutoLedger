package handlers

import "testing"

func TestDerivedFuelQuantityIsValidated(t *testing.T) {
	for _, tc := range []struct{ amount, price float64 }{{1000, 0.001}, {0.01, 10}} {
		_, err := buildFuelLog("vehicle", &SaveFuelLogRequest{Date: "2026-10-05", Amount: cents(tc.amount), PricePerLiter: f64(tc.price)})
		if err == nil || errorCode(err) != "fuel.liters_range" {
			t.Fatalf("derived quantity for %+v must be rejected: %v", tc, err)
		}
	}
	for _, liters := range []float64{0.01, 50, 500} {
		log, err := buildFuelLog("vehicle", &SaveFuelLogRequest{Date: "2026-10-05", Amount: cents(liters * 2), PricePerLiter: f64(2)})
		if err != nil || log.Liters == nil || *log.Liters != liters {
			t.Fatalf("valid derived quantity %v: %+v %v", liters, log, err)
		}
	}
}
