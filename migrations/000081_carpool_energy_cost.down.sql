UPDATE carpool_legs
SET cost_overrides = (cost_overrides - 'energy_cost') || jsonb_build_object('electricity_cost', cost_overrides -> 'energy_cost')
WHERE cost_overrides ? 'energy_cost';
UPDATE carpool_trips SET electricity_cost = energy_cost;
UPDATE carpool_legs SET electricity_cost = energy_cost;
ALTER TABLE carpool_legs DROP COLUMN fuel_cost, DROP COLUMN energy_cost;
ALTER TABLE carpool_trips DROP COLUMN fuel_cost, DROP COLUMN energy_cost;
