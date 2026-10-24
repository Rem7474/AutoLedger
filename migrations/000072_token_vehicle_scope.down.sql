ALTER TABLE oauth_codes DROP CONSTRAINT oauth_codes_vehicle_ids_not_empty;
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_vehicle_ids_not_empty;
ALTER TABLE api_tokens DROP COLUMN vehicle_ids;
ALTER TABLE oauth_codes DROP COLUMN vehicle_ids;
