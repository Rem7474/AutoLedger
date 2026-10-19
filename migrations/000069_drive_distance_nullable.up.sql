-- A drive reported without a distance (an integration with no odometer) is stored with no distance instead of 0,
-- so that an unknown distance is not read as a drive of zero kilometres. Manual and CSV drives require a distance,
-- so a 0 on a drive that is not synchronized from TeslaMate was always this placeholder; a TeslaMate drive of 0 km
-- is a measurement and stays.
ALTER TABLE drives ALTER COLUMN distance_km DROP NOT NULL;
UPDATE drives SET distance_km = NULL WHERE distance_km = 0 AND origin <> 'TESLAMATE';
