-- Optimistic locking: a save made from an out-of-date read is refused (see internal/database/versioning.go).
ALTER TABLE fuel_logs ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE maintenance_expenses ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE odometer_checkpoints ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE trip_groups ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE drive_expenses ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
