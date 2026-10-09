-- Refuse a downgrade if any price exceeds the old range rather than discard recorded data.
ALTER TABLE fuel_logs ALTER COLUMN price_per_liter TYPE NUMERIC(6, 3);
