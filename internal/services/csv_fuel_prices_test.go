package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestCSVFuelPriceRangeMessageUsesDecimalBound(t *testing.T) {
	for _, csv := range []string{
		fmt.Sprintf("date,liters,price_per_liter\n2026-10-05,1,%.3f\n", models.MaxFuelPricePerLiter+1),
		"date,liters,amount\n2026-10-05,0.01,20000000\n",
	} {
		res := previewOf(t, testVehicle(models.PowertrainICE), csv, CSVImportOptions{})
		if len(res.Errors) != 1 || res.Errors[0].Message != "Line 2: invalid price per litre (0 to 999999999.999)" {
			t.Fatalf("unexpected CSV price error: %+v", res.Errors)
		}
		if res.Errors[0].Params["p1"] != models.MaxFuelPricePerLiter {
			t.Fatalf("translation parameter changed: %v", res.Errors[0].Params)
		}
	}
}

func TestCSVFuelPricesSupportLargerCurrencyValues(t *testing.T) {
	for _, csv := range []string{
		"date,liters,price_per_liter\n2026-10-05,50,1400\n",
		"date,liters,amount\n2026-10-05,50,70000\n",
		"date,liters,price_per_liter,amount\n2026-10-05,50,1400,70000\n",
	} {
		res := previewOf(t, testVehicle(models.PowertrainICE), csv, CSVImportOptions{})
		if res.ValidRows != 1 || res.InvalidRows != 0 {
			t.Fatalf("preview: %+v", res)
		}
	}
	for _, csv := range []string{
		"date,liters,price_per_liter\n2026-10-05,500,999999999\n",
		"date,liters,amount\n2026-10-05,0.01,20000000\n",
	} {
		res := previewOf(t, testVehicle(models.PowertrainICE), csv, CSVImportOptions{})
		if res.ValidRows != 0 || res.InvalidRows != 1 {
			t.Fatalf("invalid computed amount or price accepted: %+v", res)
		}
	}
}

func TestCSVFuelPricesPersistWithoutOverflow(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	v := mustVehicle(t, repo, "fuel-csv-currency@example.org")
	v.Powertrain = models.PowertrainICE
	if err := repo.UpdateVehicle(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	svc := NewCSVImportService(repo, "Europe/Paris")
	res, err := svc.Execute(context.Background(), v, []byte("date,liters,price_per_liter\n2026-10-05,50,1400\n"), CSVImportOptions{})
	if err != nil || !res.Committed || res.ImportedCount != 1 {
		t.Fatalf("execute: %+v %v", res, err)
	}
	logs, err := repo.ListFuelLogs(context.Background(), v.ID)
	if err != nil || len(logs) != 1 || logs[0].PricePerLiter == nil || *logs[0].PricePerLiter != 1400 {
		t.Fatalf("stored: %+v %v", logs, err)
	}
}
