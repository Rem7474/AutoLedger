-- A toll invoice replaces, for the period it covers, every individual toll of that vehicle in the
-- monthly and global totals. Trips, carpools and trip groups keep their own entered tolls.
--
-- The invoice covers an explicit period [period_start, period_end) rather than a calendar month
-- key, so the view and every aggregate are indifferent to how many invoices a vehicle has or how
-- long a period is. "One invoice per vehicle and month" is a single unique index.
CREATE TABLE toll_invoices (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id   UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    month        TEXT NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    total        NUMERIC(12,2) NOT NULL CHECK (total >= 0),
    note         TEXT,
    version      INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (period_end > period_start)
);
CREATE UNIQUE INDEX toll_invoices_vehicle_month_key ON toll_invoices (vehicle_id, month);
CREATE INDEX toll_invoices_vehicle_period ON toll_invoices (vehicle_id, period_start, period_end);

CREATE OR REPLACE VIEW cost_ledger AS
-- Energy
SELECT c.vehicle_id,
       c.date AS entry_date,
       'ENERGY'::text AS category,
       'charge_logs'::text AS source_table,
       c.id AS source_id,
       ROUND(CASE WHEN c.currency = v.currency THEN c.cost ELSE c.cost * c.fx_rate END, 2) AS amount_eur
FROM charge_logs c
JOIN vehicles v ON v.id = c.vehicle_id
WHERE c.deleted_upstream_at IS NULL
  AND c.cost IS NOT NULL
  AND (c.currency = v.currency OR c.fx_rate IS NOT NULL)

UNION ALL
-- Energy: fuel fill-ups (combustion vehicles)
SELECT f.vehicle_id,
       f.date AS entry_date,
       'ENERGY'::text,
       'fuel_logs'::text,
       f.id,
       ROUND(f.amount, 2)
FROM fuel_logs f

UNION ALL
-- Travel expenses (tolls, parking, ferries)
SELECT e.vehicle_id,
       e.date,
       CASE e.type WHEN 'TOLL' THEN 'TOLL' WHEN 'PARKING' THEN 'PARKING' ELSE 'TRAVEL_OTHER' END,
       'drive_expenses',
       e.id,
       ROUND(CASE WHEN e.currency = v.currency THEN e.amount ELSE e.amount * e.fx_rate END, 2)
FROM drive_expenses e
JOIN vehicles v ON v.id = e.vehicle_id
WHERE (e.currency = v.currency OR e.fx_rate IS NOT NULL)
  -- A month with a toll invoice is accounted for by the invoice alone (see below).
  AND NOT (e.type = 'TOLL' AND EXISTS (
      SELECT 1 FROM toll_invoices i
      WHERE i.vehicle_id = e.vehicle_id AND e.date >= i.period_start AND e.date < i.period_end))

UNION ALL
-- Toll invoices: one TOLL line per invoice, dated at the start of the period it covers
SELECT i.vehicle_id,
       i.period_start,
       'TOLL'::text,
       'toll_invoices'::text,
       i.id,
       i.total
FROM toll_invoices i

UNION ALL
-- Maintenance, insurance and fixed expenses, recurring ones expanded
SELECT m.vehicle_id,
       occ,
       m.category,
       'maintenance_expenses',
       m.id,
       ROUND(CASE WHEN m.currency = v.currency THEN m.amount ELSE m.amount * m.fx_rate END, 2)
FROM maintenance_expenses m
JOIN vehicles v ON v.id = m.vehicle_id
LEFT JOIN vehicle_ownership o ON o.vehicle_id = m.vehicle_id
CROSS JOIN LATERAL generate_series(
    m.date,
    CASE
        WHEN m.is_recurring AND COALESCE(m.recurrence_interval_months, 0) > 0
        THEN GREATEST(m.date, LEAST(NOW(), COALESCE(m.recurrence_end_date, 'infinity'::timestamptz),
                                    COALESCE((o.end_date + 1)::timestamptz - INTERVAL '1 microsecond', 'infinity'::timestamptz)))
        ELSE m.date
    END,
    make_interval(months => GREATEST(COALESCE(m.recurrence_interval_months, 1), 1))
) AS occ
WHERE m.currency = v.currency OR m.fx_rate IS NOT NULL

UNION ALL
-- Tire purchases
SELECT t.vehicle_id,
       t.purchase_date::timestamptz,
       'TIRES',
       'tires',
       t.id,
       t.purchase_price
FROM tires t
WHERE t.vehicle_id IS NOT NULL AND t.purchase_price > 0

UNION ALL
-- Purchase (cash or loan): price and acquisition fees, net of incentives
SELECT o.vehicle_id,
       o.start_date::timestamptz,
       'ACQUISITION',
       'vehicle_ownership',
       o.vehicle_id,
       o.purchase_price + COALESCE(o.purchase_fees, 0) - COALESCE(o.incentives, 0)
FROM vehicle_ownership o
WHERE o.acquisition_type IN ('CASH', 'LOAN') AND o.purchase_price > 0

UNION ALL
-- LOA purchase option exercised
SELECT o.vehicle_id,
       o.option_exercised_date::timestamptz,
       'ACQUISITION',
       'vehicle_ownership',
       o.vehicle_id,
       o.lease_purchase_option_price
FROM vehicle_ownership o
WHERE o.acquisition_type = 'LOA' AND o.option_exercised_date IS NOT NULL
  AND o.lease_purchase_option_price > 0 AND o.option_exercised_date <= CURRENT_DATE

UNION ALL
-- One-off financing fees: loan fees, lease down payment and application fees at contract start
SELECT o.vehicle_id,
       o.start_date::timestamptz,
       'FINANCING',
       'vehicle_ownership',
       o.vehicle_id,
       CASE WHEN o.acquisition_type = 'LOAN' THEN COALESCE(o.loan_fees, 0)
            ELSE COALESCE(o.lease_down_payment, 0) + COALESCE(o.lease_fees, 0) END
FROM vehicle_ownership o
WHERE o.start_date <= CURRENT_DATE
  AND ((o.acquisition_type = 'LOAN' AND o.loan_fees > 0)
    OR (o.acquisition_type IN ('LOA', 'LLD') AND COALESCE(o.lease_down_payment, 0) + COALESCE(o.lease_fees, 0) > 0))

UNION ALL
-- Loan monthly interest (annuity schedule, first payment one month after start) and borrower insurance
SELECT o.vehicle_id,
       (o.start_date + make_interval(months => k))::timestamptz,
       'FINANCING',
       'vehicle_ownership',
       o.vehicle_id,
       ROUND(
           CASE WHEN COALESCE(o.loan_rate_pct, 0) = 0 THEN 0
                ELSE (o.loan_amount * power(1 + o.loan_rate_pct / 1200, k - 1)
                      - (o.loan_amount * (o.loan_rate_pct / 1200) / (1 - power(1 + o.loan_rate_pct / 1200, -o.loan_duration_months)))
                        * (power(1 + o.loan_rate_pct / 1200, k - 1) - 1) / (o.loan_rate_pct / 1200))
                     * (o.loan_rate_pct / 1200)
           END + COALESCE(o.loan_insurance_monthly, 0),
       2)
FROM vehicle_ownership o
CROSS JOIN LATERAL generate_series(1, o.loan_duration_months) AS k
WHERE o.acquisition_type = 'LOAN'
  AND o.loan_amount > 0 AND o.loan_duration_months > 0
  AND o.start_date + make_interval(months => k) <= LEAST(CURRENT_DATE, COALESCE(o.end_date, CURRENT_DATE))

UNION ALL
-- Lease monthly rents (first rent at contract start), until the contract ends, the vehicle is returned or the option is exercised
SELECT o.vehicle_id,
       (o.start_date + make_interval(months => k))::timestamptz,
       'FINANCING',
       'vehicle_ownership',
       o.vehicle_id,
       o.lease_monthly_rent
FROM vehicle_ownership o
CROSS JOIN LATERAL generate_series(0, o.lease_duration_months - 1) AS k
WHERE o.acquisition_type IN ('LOA', 'LLD')
  AND o.lease_monthly_rent > 0 AND o.lease_duration_months > 0
  AND o.start_date + make_interval(months => k)
      < LEAST(CURRENT_DATE + 1, COALESCE(o.end_date, 'infinity'::date), COALESCE(o.option_exercised_date, 'infinity'::date))

UNION ALL
-- Lease return fees, once the contract is over without exercising the option
SELECT o.vehicle_id,
       LEAST(COALESCE(o.end_date, 'infinity'::date), (o.start_date + make_interval(months => o.lease_duration_months))::date)::timestamptz,
       'FINANCING',
       'vehicle_ownership',
       o.vehicle_id,
       o.lease_end_fees_estimate
FROM vehicle_ownership o
WHERE o.acquisition_type IN ('LOA', 'LLD')
  AND o.lease_end_fees_estimate > 0
  AND o.option_exercised_date IS NULL
  AND LEAST(COALESCE(o.end_date, 'infinity'::date), (o.start_date + make_interval(months => o.lease_duration_months))::date) <= CURRENT_DATE;
