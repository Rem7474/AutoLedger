-- A grant given to an assistant can be limited to some vehicles of the account. NULL means every vehicle the account
-- can see, which is what every token had until now.
ALTER TABLE oauth_codes ADD COLUMN vehicle_ids UUID[];
ALTER TABLE api_tokens ADD COLUMN vehicle_ids UUID[];
ALTER TABLE api_tokens ADD CONSTRAINT api_tokens_vehicle_ids_not_empty CHECK (vehicle_ids IS NULL OR cardinality(vehicle_ids) > 0);
ALTER TABLE oauth_codes ADD CONSTRAINT oauth_codes_vehicle_ids_not_empty CHECK (vehicle_ids IS NULL OR cardinality(vehicle_ids) > 0);
