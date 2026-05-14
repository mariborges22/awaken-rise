-- 000002_expand_tenant_business_info.down.sql

DROP INDEX idx_tenants_cnpj ON tenants;

ALTER TABLE tenants 
DROP COLUMN legal_name,
DROP COLUMN cnpj,
DROP COLUMN contact_email,
DROP COLUMN contact_phone,
DROP COLUMN plan_type,
DROP COLUMN verification_status,
DROP COLUMN monthly_usage_count;
