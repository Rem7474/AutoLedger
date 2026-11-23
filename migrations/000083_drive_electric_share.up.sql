-- The electric share of a manual drive of a plug-in hybrid or range extender, frozen when the drive is saved from the
-- share its owner had accepted: a later change of the vehicle's share never reprices a drive already recorded.
-- NULL: no split (a drive of another powertrain, a synchronized drive, or a drive saved before a share was accepted).
ALTER TABLE drives ADD COLUMN electric_share NUMERIC(4, 3)
    CHECK (electric_share IS NULL OR (electric_share >= 0 AND electric_share <= 1));
-- Manual drives of hybrids saved so far were priced entirely on electricity: they keep that split.
UPDATE drives SET electric_share = 1
WHERE is_manual AND electric_share IS NULL
  AND vehicle_id IN (SELECT id FROM vehicles WHERE powertrain IN ('PHEV', 'REEV'));
