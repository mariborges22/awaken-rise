-- 000002_expand_tenant_business_info.up.sql

ALTER TABLE tenants 
ADD COLUMN legal_name VARCHAR(255) AFTER tenant_id,
ADD COLUMN cnpj VARCHAR(18) UNIQUE AFTER legal_name,
ADD COLUMN contact_email VARCHAR(150) AFTER mp_access_token,
ADD COLUMN contact_phone VARCHAR(20) AFTER contact_email,
ADD COLUMN plan_type VARCHAR(50) DEFAULT 'starter' AFTER status,
ADD COLUMN verification_status VARCHAR(50) DEFAULT 'pending' AFTER plan_type,
ADD COLUMN monthly_usage_count INT DEFAULT 0 AFTER verification_status;

-- Index para busca rápida por CNPJ (Compliance)
CREATE INDEX idx_tenants_cnpj ON tenants(cnpj);
