package services

import (
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestCSVRejectsInvalidDerivedFuelQuantity(t *testing.T) {
	for _, row := range []string{"2026-10-05,1000,0.001", "2026-10-05,0.01,10"} {
		res := previewOf(t, testVehicle(models.PowertrainICE), "date,amount,price_per_liter\n"+row+"\n", CSVImportOptions{})
		if res.ValidRows != 0 || len(res.Errors) != 1 || res.Errors[0].Code != "import.row.invalid_liters" {
			t.Fatalf("invalid derived quantity accepted: %+v", res)
		}
	}
	res := previewOf(t, testVehicle(models.PowertrainICE), "date,amount,price_per_liter\n2026-10-05,100,2\n", CSVImportOptions{})
	if res.ValidRows != 1 || res.InvalidRows != 0 {
		t.Fatalf("valid derived quantity rejected: %+v", res)
	}
}
