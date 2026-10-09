package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type supplierOAuthProxyRepoStub struct {
	service.ProxyRepository
}

func (supplierOAuthProxyRepoStub) ListActive(context.Context) ([]service.Proxy, error) {
	return nil, nil
}

func TestSupplierOAuthURLCanBeCopiedFromRawJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	supplierID := int64(7)
	supplier := &service.Supplier{
		ID: supplierID, Status: domain.SupplierStatusActive,
		AllowedAccountKinds: []service.SupplierAccountKind{{Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth}},
	}
	accounts := service.NewSupplierAccountService(nil,
		service.NewSupplierService(&supplierTestSupplierRepository{supplier: supplier}), nil,
		service.NewProxyService(supplierOAuthProxyRepoStub{}), nil, nil)
	handler := &SupplierHandler{accounts: accounts, openAIOAuth: service.NewOpenAIOAuthService(nil, nil)}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(middleware.ContextKeySupplierID), supplierID); c.Next() })
	router.POST("/supplier/oauth/:platform/auth-url", handler.GenerateSupplierOAuthURL)
	request := httptest.NewRequest(http.MethodPost, "/supplier/oauth/openai/auth-url", strings.NewReader(`{"proxy_id":0,"type":"oauth"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "&code_challenge=")
	require.NotContains(t, recorder.Body.String(), `\u0026`)
	var payload struct {
		Code int                         `json:"code"`
		Data service.OpenAIAuthURLResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Zero(t, payload.Code)
	require.NotEmpty(t, payload.Data.SessionID)
	require.Contains(t, payload.Data.AuthURL, "&code_challenge=")
}
