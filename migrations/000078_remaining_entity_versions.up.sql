-- Optimistic locking for tires, mount sessions, tread measurements, reminders, tariff plans and comparison scenarios.
ALTER TABLE tires ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tire_mount_sessions ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tire_logs ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE maintenance_reminders ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tariff_plans ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE comparison_scenarios ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE vehicle_ownership ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
