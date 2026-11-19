-- What an owner accepted of the electric share suggested for a plug-in hybrid or range extender. The suggestion
-- itself is never stored: only the value the owner accepted (or typed), which stays as it is until they change it.
ALTER TABLE vehicles ADD COLUMN hybrid_electric_share NUMERIC(4, 3)
    CHECK (hybrid_electric_share IS NULL OR (hybrid_electric_share >= 0 AND hybrid_electric_share <= 1));
-- The fuel consumption on the kilometres driven on fuel, accepted with the share (litres per 100 km).
ALTER TABLE vehicles ADD COLUMN hybrid_fuel_l_100km NUMERIC(6, 2)
    CHECK (hybrid_fuel_l_100km IS NULL OR hybrid_fuel_l_100km > 0);
-- Optional correction of the usable battery capacity, otherwise deduced from the charges that carry percentages.
ALTER TABLE vehicles ADD COLUMN battery_usable_kwh NUMERIC(6, 2)
    CHECK (battery_usable_kwh IS NULL OR battery_usable_kwh > 0);
