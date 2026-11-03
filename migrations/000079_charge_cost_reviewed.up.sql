-- A synchronised charge priced 0 is either a free charge or a cost TeslaMate does not know yet: the source reports
-- both as 0. The owner says which with cost_reviewed_at (the same idea as drives.toll_reviewed_at).
ALTER TABLE charge_logs ADD COLUMN IF NOT EXISTS cost_reviewed_at TIMESTAMPTZ;

-- A charge older than the 30 days the synchronisation re-reads cannot get its cost from TeslaMate any more: its 0 stands.
UPDATE charge_logs
SET cost_reviewed_at = NOW()
WHERE origin = 'TESLAMATE' AND cost_source = 'TESLAMATE' AND cost = 0
  AND cost_reviewed_at IS NULL AND date < NOW() - INTERVAL '30 days';
