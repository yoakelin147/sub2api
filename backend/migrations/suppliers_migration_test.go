package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuppliersMigrationAddsTenantOwnershipAndReviewState(t *testing.T) {
	content, err := FS.ReadFile("239_add_suppliers.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS suppliers")
	require.Contains(t, sql, "allowed_account_kinds JSONB NOT NULL DEFAULT '[]'::jsonb")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS supplier_id BIGINT REFERENCES suppliers(id)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS supplier_external_id VARCHAR(191)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'approved'")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_supplier_external_id")
	require.Contains(t, sql, "WHERE supplier_id IS NOT NULL AND supplier_external_id IS NOT NULL AND deleted_at IS NULL")
	require.Contains(t, sql, "users_supplier_role_check")
	require.Contains(t, sql, "role = 'supplier' AND supplier_id IS NOT NULL")
}
