-- Fuel prices retain three decimals and support currencies with larger nominal unit prices.
ALTER TABLE fuel_logs ALTER COLUMN price_per_liter TYPE NUMERIC(12, 3);
