-- 000003_add_payment_hub_config.down.sql

ALTER TABLE tenants 
CHANGE COLUMN encrypted_config mp_access_token TEXT,
DROP COLUMN payment_provider;
