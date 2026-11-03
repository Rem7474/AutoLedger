-- A counter the user-facing writes advance, so that a save made from an out-of-date read can be refused.
ALTER TABLE carpool_trips ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
