package admin

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAdminSupplierResponseDoesNotExposeTokenHashOrSelector(t *testing.T) {
	hash, selector, prefix := "SECRET-HASH", "SECRET-SELECTOR", "supplier_abc"
	payload, err := json.Marshal(adminSupplierResponse(&service.Supplier{
		ID: 1, Code: "demo", Name: "Demo", TokenHash: &hash, TokenSelector: &selector, TokenPrefix: &prefix,
	}))
	require.NoError(t, err)
	require.NotContains(t, string(payload), hash)
	require.NotContains(t, string(payload), selector)
	require.Contains(t, string(payload), prefix)
}
