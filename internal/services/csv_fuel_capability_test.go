package services

import (
	"context"
	"testing"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

func TestFuelImportRequiresRefuelingCapability(t *testing.T) {
	svc := NewCSVImportService(nil, "UTC")
	csv := []byte("date,liters,price_per_liter\n2026-10-05,50,2\n")
	for _, kind := range []ImportType{"", ImportTypeFuel} {
		opts := CSVImportOptions{Type: kind}
		for _, execute := range []bool{false, true} {
			var err error
			if execute {
				_, err = svc.Execute(context.Background(), testVehicle(models.PowertrainEV), csv, opts)
			} else {
				_, err = svc.Preview(context.Background(), testVehicle(models.PowertrainEV), csv, opts)
			}
			apiErr, ok := apierror.As(err)
			if !ok || apiErr.Code != "import.ev_fuel" {
				t.Fatalf("EV fuel import must fail before persistence: %v", err)
			}
		}
		for _, powertrain := range []string{models.PowertrainICE, models.PowertrainPHEV, models.PowertrainREEV} {
			res, err := svc.Preview(context.Background(), testVehicle(powertrain), csv, opts)
			if err != nil || res.ValidRows != 1 {
				t.Fatalf("%s fuel import rejected: %+v %v", powertrain, res, err)
			}
		}
	}
}
