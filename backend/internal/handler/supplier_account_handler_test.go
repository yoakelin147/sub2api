package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountResponseNeverExposesCredentialsOrExtra(t *testing.T) {
	supplierID := int64(7)
	externalID := "vendor-1"
	account := &service.Account{
		ID: 11, SupplierID: &supplierID, SupplierExternalID: &externalID,
		Name: "account", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "SECRET-ACCESS", "refresh_token": "SECRET-REFRESH"},
		Extra:       map[string]any{"private": "SECRET-EXTRA"},
	}

	payload, err := json.Marshal(supplierAccountResponse(account))
	require.NoError(t, err)
	require.NotContains(t, string(payload), "SECRET-ACCESS")
	require.NotContains(t, string(payload), "SECRET-REFRESH")
	require.NotContains(t, string(payload), "SECRET-EXTRA")
	require.Contains(t, string(payload), "vendor-1")
}

func TestDecodeStrictSupplierJSONRejectsAdminFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/api/v1/supplier/accounts", bytes.NewBufferString(`{
		"name":"account","platform":"openai","type":"apikey",
		"credentials":{"api_key":"secret"},"group_ids":[1]
	}`))
	var request supplierAccountRequest
	require.Error(t, decodeStrictSupplierJSON(ctx, &request))
}
