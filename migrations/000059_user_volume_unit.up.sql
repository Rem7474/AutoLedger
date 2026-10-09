-- ============================================================================
-- AutoLedger User Volume Unit Migration (Up)
-- Database: PostgreSQL 14+
-- ============================================================================

-- Fuel volumes are always stored and computed in litres; this only controls what the frontend
-- converts to for display and form input, next to distance_unit. Per account, like language.
ALTER TABLE users ADD COLUMN volume_unit TEXT NOT NULL DEFAULT 'l' CHECK (volume_unit IN ('l', 'gal_us', 'gal_uk'));
