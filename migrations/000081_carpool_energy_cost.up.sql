-- The energy of a carpool is named for what it is, whatever the energy: energy_cost is the total, fuel_cost its fuel
-- part, and the electric part stays in electricity_cost (kept so that nothing reading it breaks). A vehicle that burns
-- fuel only used to store its fuel in electricity_cost: it moves to fuel_cost.
ALTER TABLE carpool_trips ADD COLUMN energy_cost NUMERIC(10, 2);
ALTER TABLE carpool_trips ADD COLUMN fuel_cost NUMERIC(10, 2) NOT NULL DEFAULT 0;
ALTER TABLE carpool_legs ADD COLUMN energy_cost NUMERIC(10, 2);
ALTER TABLE carpool_legs ADD COLUMN fuel_cost NUMERIC(10, 2) NOT NULL DEFAULT 0;

UPDATE carpool_trips t SET energy_cost = t.electricity_cost,
    fuel_cost = CASE WHEN v.powertrain = 'ICE' THEN t.electricity_cost ELSE 0 END,
    electricity_cost = CASE WHEN v.powertrain = 'ICE' THEN 0 ELSE t.electricity_cost END
FROM vehicles v WHERE v.id = t.vehicle_id;
UPDATE carpool_trips SET energy_cost = electricity_cost WHERE energy_cost IS NULL;

UPDATE carpool_legs l SET energy_cost = l.electricity_cost,
    fuel_cost = CASE WHEN v.powertrain = 'ICE' THEN l.electricity_cost ELSE 0 END,
    electricity_cost = CASE WHEN v.powertrain = 'ICE' THEN 0 ELSE l.electricity_cost END
FROM carpool_trips t JOIN vehicles v ON v.id = t.vehicle_id WHERE t.id = l.carpool_trip_id;
UPDATE carpool_legs SET energy_cost = electricity_cost WHERE energy_cost IS NULL;

ALTER TABLE carpool_trips ALTER COLUMN energy_cost SET NOT NULL, ALTER COLUMN energy_cost SET DEFAULT 0;
ALTER TABLE carpool_legs ALTER COLUMN energy_cost SET NOT NULL, ALTER COLUMN energy_cost SET DEFAULT 0;

-- The categories a leg overrides are named by their field: electricity_cost becomes energy_cost.
UPDATE carpool_legs
SET cost_overrides = (cost_overrides - 'electricity_cost') || jsonb_build_object('energy_cost', cost_overrides -> 'electricity_cost')
WHERE cost_overrides ? 'electricity_cost';
