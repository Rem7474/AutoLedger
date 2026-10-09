package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

func TestFuelPriceRangeMessageUsesDecimalBound(t *testing.T) {
	err := validateFuelPrice(f64(models.MaxFuelPricePerLiter + 1))
	apiErr, ok := apierror.As(err)
	if !ok || apiErr.Message != "Invalid price per litre (0 to 999999999.999)" {
		t.Fatalf("unexpected price range message: %v", err)
	}
	if apiErr.Params["p0"] != models.MaxFuelPricePerLiter {
		t.Fatalf("translation parameter changed: %v", apiErr.Params)
	}
}

func TestBuildFuelLogSupportsLargerNominalPrices(t *testing.T) {
	for _, price := range []float64{200, 1400, 12500} {
		f, err := buildFuelLog("v", &SaveFuelLogRequest{Date: "2026-10-05", Liters: f64(50), PricePerLiter: f64(price)})
		if err != nil || f.PricePerLiter == nil || *f.PricePerLiter != price || f.Amount.Float() != 50*price {
			t.Fatalf("price=%v: %+v %v", price, f, err)
		}
		_, err = buildFuelLog("v", &SaveFuelLogRequest{Date: "2026-10-05", Liters: f.Liters, Amount: &f.Amount, PricePerLiter: f.PricePerLiter})
		if err != nil {
			t.Fatalf("editing price %v: %v", price, err)
		}
	}
	_, err := buildFuelLog("v", &SaveFuelLogRequest{Date: "2026-10-05", Liters: f64(0.01), Amount: cents(20000000)})
	if err == nil || errorCode(err) != "fuel.price_range" {
		t.Fatalf("derived price above storage limit: %v", err)
	}
}

func TestFuelPriceStorageSupportsFourDigitPrices(t *testing.T) {
	a := newVehicleAPI(t, "fuel-price-currency")
	a.router.Post("/{vehicleId}/fuel", NewFuelHandler(a.repo).Create)
	a.router.Put("/{vehicleId}/fuel/{fuelLogId}", NewFuelHandler(a.repo).Update)
	created := decodeObj(t, a.do(a.ownerID, 201, "POST", "/", `{"name":"Combustion","powertrain":"ICE","currency":"ARS"}`))
	vid := created["id"].(string)
	for _, body := range []string{
		`{"date":"2026-10-05","liters":50,"amount":70000}`,
		`{"date":"2026-10-06","liters":50,"price_per_liter":1400}`,
	} {
		f := decodeObj(t, a.do(a.ownerID, 201, "POST", "/"+vid+"/fuel", body))
		if f["price_per_liter"] != float64(1400) || f["amount"] != float64(70000) {
			t.Fatalf("stored: %v", f)
		}
		a.do(a.ownerID, 200, "PUT", "/"+vid+"/fuel/"+f["id"].(string), `{"date":"2026-10-05","liters":50,"amount":70000,"price_per_liter":1400,"notes":"Receipt added"}`)
	}
	// The bound still applies to explicit and derived prices before PostgreSQL is called.
	a.do(a.ownerID, 400, "POST", "/"+vid+"/fuel", `{"date":"2026-10-07","liters":0.01,"amount":20000000}`)
}
