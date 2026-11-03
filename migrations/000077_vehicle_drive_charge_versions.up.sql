-- Optimistic locking for vehicles, drives and charges. Sync and geocoding jobs never advance the version.
ALTER TABLE vehicles ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE drives ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE charge_logs ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
