-- 000005_auth_catalog_billing.down.sql

ALTER TABLE tenants 
DROP COLUMN subscription_expires_at,
DROP COLUMN subscription_status,
DROP COLUMN subscription_id;

DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;
