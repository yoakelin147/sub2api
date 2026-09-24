package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupplierCreateAccountRequiresIdempotencyKey(t *testing.T) {
	previous := service.DefaultIdempotencyCoordinator()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newUserMemoryIdempotencyRepoStub(), service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(previous) })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(string(middleware.ContextKeySupplierID), int64(7))
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/supplier/accounts", bytes.NewBufferString(`{
		"external_id":"vendor-1","name":"account","platform":"openai","type":"apikey",
		"credentials":{"api_key":"secret"}
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	(&SupplierHandler{}).CreateAccount(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "IDEMPOTENCY_KEY_REQUIRED")
}

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

func TestSupplierProfileOnlyExposesAuthorizedGroupsWhenAvailable(t *testing.T) {
	supplier := &service.Supplier{ReviewRequired: true, AutoApproveGroups: map[string][]int64{"openai": {42}}}
	require.Equal(t, []service.SupplierAccountKind{}, supplierResponse(supplier)["allowed_account_kinds"])
	require.Empty(t, supplierResponse(supplier)["auto_approve_groups"])
	supplier.ReviewRequired = false
	require.Equal(t, supplier.AutoApproveGroups, supplierResponse(supplier)["auto_approve_groups"])
}

func TestDecodeStrictSupplierJSONRejectsAdminFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/api/v1/supplier/accounts", bytes.NewBufferString(`{
		"name":"account","platform":"openai","type":"apikey",
		"credentials":{"api_key":"secret"},"rate_multiplier":2
	}`))
	var request supplierAccountRequest
	require.Error(t, decodeStrictSupplierJSON(ctx, &request))
}

func TestSupplierBatchCreateReturns413ForOversizedBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := `{"accounts":[]}` + strings.Repeat(" ", maxSupplierAccountBatchRequestBytes)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/supplier/accounts/batch", strings.NewReader(body))

	(&SupplierHandler{}).BatchCreateAccounts(ctx)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), "BATCH_TOO_LARGE")
}

func TestDecodeStrictSupplierJSONRejectsBodyOverLimitAfterValidJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := `{"name":"account","platform":"openai","type":"apikey","credentials":{"api_key":"secret"}}` +
		strings.Repeat(" ", maxSupplierAccountRequestBytes)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/supplier/accounts", strings.NewReader(body))

	var request supplierAccountRequest
	require.ErrorIs(t, decodeStrictSupplierJSON(ctx, &request), errSupplierRequestTooLarge)
}
