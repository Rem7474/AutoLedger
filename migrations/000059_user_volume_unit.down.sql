-- ============================================================================
-- AutoLedger User Volume Unit Migration (Down)
-- ============================================================================

ALTER TABLE users DROP COLUMN volume_unit;
