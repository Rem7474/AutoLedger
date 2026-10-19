UPDATE drives SET distance_km = 0 WHERE distance_km IS NULL;
ALTER TABLE drives ALTER COLUMN distance_km SET NOT NULL;
