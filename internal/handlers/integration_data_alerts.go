package handlers

import (
	"strings"

	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationDataAlert is one recent entry that looks mistyped. It names the day, not the record: the owner finds
// it by its date and amount.
type IntegrationDataAlert struct {
	Kind     string `json:"kind" jsonschema:"amount_outlier (an expense far above the usual of its category) or price_outlier (a price per kWh or per litre far outside the vehicle's own range)"`
	Category string `json:"category" jsonschema:"toll, parking or travel_other for an amount outlier; charge or fuel for a price outlier"`
	Date     string `json:"date" jsonschema:"day of the entry, YYYY-MM-DD"`
	// The amount outlier is compared with the median of its category.
	Amount      *IntegrationAmount `json:"amount" jsonschema:"for an amount outlier, what the entry cost"`
	UsualAmount *IntegrationAmount `json:"usual_amount" jsonschema:"for an amount outlier, the median of the category, what an entry usually costs"`
	// The price outlier is compared with the range of the vehicle's other prices.
	UnitPrice *IntegrationAmount `json:"unit_price" jsonschema:"for a price outlier, the price per kWh or per litre of the entry"`
	LowPrice  *IntegrationAmount `json:"lowest_other_price" jsonschema:"for a price outlier, the lowest price among the vehicle's other entries"`
	HighPrice *IntegrationAmount `json:"highest_other_price" jsonschema:"for a price outlier, the highest price among the vehicle's other entries"`
}

func integrationDataAlerts(alerts []services.DataAlert, currency string) []IntegrationDataAlert {
	out := make([]IntegrationDataAlert, 0, len(alerts))
	for _, a := range alerts {
		item := IntegrationDataAlert{Kind: a.Kind, Category: strings.ToLower(a.Category), Date: a.Date.Format("2006-01-02")}
		if a.Kind == services.AlertAmountOutlier {
			amount, usual := amountOf(a.Amount, currency), amountOf(a.Median, currency)
			item.Amount, item.UsualAmount = &amount, &usual
		} else {
			item.UnitPrice = decimalAmount(&a.UnitPrice, currency, 3)
			item.LowPrice = decimalAmount(&a.Low, currency, 3)
			item.HighPrice = decimalAmount(&a.High, currency, 3)
		}
		out = append(out, item)
	}
	return out
}
