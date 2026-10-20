-- A carpool leg made from a drive follows the cost of that drive, with optional per-category overrides. Freezing a
-- carpool keeps its amounts as entered, so the figures already given to passengers stop moving.
ALTER TABLE carpool_legs ADD COLUMN cost_overrides JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE carpool_trips ADD COLUMN costs_frozen_at TIMESTAMPTZ;

-- Carpools saved before keep the amounts they were saved with.
UPDATE carpool_trips t
SET costs_frozen_at = t.updated_at
WHERE EXISTS (SELECT 1 FROM carpool_legs l WHERE l.carpool_trip_id = t.id AND l.drive_id IS NOT NULL);
