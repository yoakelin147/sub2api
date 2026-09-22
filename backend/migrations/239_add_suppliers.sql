CREATE TABLE IF NOT EXISTS suppliers (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(120) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    notes TEXT,
    allowed_account_kinds JSONB NOT NULL DEFAULT '[]'::jsonb,
    token_selector VARCHAR(64),
    token_hash VARCHAR(64),
    token_prefix VARCHAR(32),
    token_created_at TIMESTAMPTZ,
    token_last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT suppliers_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS suppliers_code_unique_active
    ON suppliers(code)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS suppliers_token_selector_unique
    ON suppliers(token_selector)
    WHERE token_selector IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_suppliers_status ON suppliers(status);
CREATE INDEX IF NOT EXISTS idx_suppliers_deleted_at ON suppliers(deleted_at);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS supplier_id BIGINT REFERENCES suppliers(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_users_supplier_id ON users(supplier_id);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_supplier_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_supplier_role_check CHECK (
        (role = 'supplier' AND supplier_id IS NOT NULL)
        OR (role IN ('admin', 'user') AND supplier_id IS NULL)
    );

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS supplier_id BIGINT REFERENCES suppliers(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS supplier_external_id VARCHAR(191),
    ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'approved',
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS review_note TEXT;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_review_status_check;
ALTER TABLE accounts
    ADD CONSTRAINT accounts_review_status_check
        CHECK (review_status IN ('pending', 'approved', 'rejected'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_supplier_external_id
    ON accounts(supplier_id, supplier_external_id)
    WHERE supplier_id IS NOT NULL AND supplier_external_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_supplier_status
    ON accounts(supplier_id, status);

CREATE INDEX IF NOT EXISTS idx_accounts_supplier_review_status
    ON accounts(supplier_id, review_status);
