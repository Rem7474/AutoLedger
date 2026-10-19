-- The household budget is an amount in the currency of the first vehicle at the time it was set. Storing the
-- currency with it keeps the figure from changing meaning when the first vehicle changes.
ALTER TABLE users ADD COLUMN fleet_monthly_budget_currency TEXT CHECK (fleet_monthly_budget_currency ~ '^[A-Z]{3}$');

UPDATE users u
SET fleet_monthly_budget_currency = COALESCE(
    (SELECT v.currency
     FROM vehicles v
     LEFT JOIN vehicle_members vm ON vm.vehicle_id = v.id AND vm.user_id = u.id
     WHERE v.user_id = u.id OR vm.user_id IS NOT NULL
     ORDER BY v.created_at, v.id
     LIMIT 1), 'EUR')
WHERE u.fleet_monthly_budget IS NOT NULL;

ALTER TABLE users ADD CONSTRAINT users_fleet_budget_has_currency
    CHECK ((fleet_monthly_budget IS NULL) = (fleet_monthly_budget_currency IS NULL));
