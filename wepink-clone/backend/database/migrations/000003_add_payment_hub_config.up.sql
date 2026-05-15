-- 000003_add_payment_hub_config.up.sql

ALTER TABLE tenants 
CHANGE COLUMN mp_access_token encrypted_config TEXT,
ADD COLUMN payment_provider VARCHAR(50) AFTER cnpj;
