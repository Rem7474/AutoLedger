package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// Idempotency-Key storage for safely replaying queued offline mutations.
// GetIdempotentResponse returns the stored response of a key, or nil.
func (r *Repository) GetIdempotentResponse(ctx context.Context, userID, key string) (*StoredResponse, error) {
	var res StoredResponse
	err := r.pool.QueryRow(ctx, `
		SELECT method, path, status_code, response_body FROM idempotency_keys WHERE user_id = $1 AND key = $2;
	`, userID, key).Scan(&res.Method, &res.Path, &res.StatusCode, &res.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SaveIdempotentResponse stores the response of a key (first writer wins) and purges keys older than 30 days.
func (r *Repository) SaveIdempotentResponse(ctx context.Context, userID, key string, res StoredResponse) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, key, method, path, status_code, response_body)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, key) DO NOTHING;
	`, userID, key, res.Method, res.Path, res.StatusCode, res.Body); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '30 days';`)
	return err
}

// ============================================================================
// Vehicle Ownership
// ============================================================================

const ownershipColumns = `
	vehicle_id, acquisition_type, start_date, start_odometer,
	purchase_price, purchase_fees, incentives, expected_resale_value, expected_holding_months,
	loan_amount, loan_rate_pct, loan_duration_months, loan_fees, loan_insurance_monthly,
	lease_down_payment, lease_monthly_rent, lease_duration_months, lease_fees, lease_deposit,
	lease_km_allowance_per_year, lease_excess_km_price, lease_end_fees_estimate, lease_purchase_option_price,
	lease_includes_maintenance, lease_includes_insurance, lease_includes_tires, option_exercised_date,
	end_date, sale_price, created_at, updated_at, version
`

func scanOwnership(row pgx.Row, o *models.VehicleOwnership) error {
	return row.Scan(
		&o.VehicleID, &o.AcquisitionType, &o.StartDate, &o.StartOdometer,
		&o.PurchasePrice, &o.PurchaseFees, &o.Incentives, &o.ExpectedResaleValue, &o.ExpectedHoldingMonths,
		&o.LoanAmount, &o.LoanRatePct, &o.LoanDurationMonths, &o.LoanFees, &o.LoanInsuranceMonthly,
		&o.LeaseDownPayment, &o.LeaseMonthlyRent, &o.LeaseDurationMonths, &o.LeaseFees, &o.LeaseDeposit,
		&o.LeaseKmAllowancePerYear, &o.LeaseExcessKmPrice, &o.LeaseEndFeesEstimate, &o.LeasePurchaseOptionPrice,
		&o.LeaseIncludesMaintenance, &o.LeaseIncludesInsurance, &o.LeaseIncludesTires, &o.OptionExercisedDate,
		&o.EndDate, &o.SalePrice, &o.CreatedAt, &o.UpdatedAt, &o.Version,
	)
}
