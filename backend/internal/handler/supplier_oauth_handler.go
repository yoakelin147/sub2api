package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *SupplierHandler) GenerateSupplierOAuthURL(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request struct {
		ProxyID   int64  `json:"proxy_id"`
		Type      string `json:"type"`
		OAuthType string `json:"oauth_type"`
		ProjectID string `json:"project_id"`
		TierID    string `json:"tier_id"`
	}
	if err := decodeStrictSupplierJSON(c, &request); err != nil || request.ProxyID <= 0 {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	platform := c.Param("platform")
	accountType := request.Type
	if accountType == "" {
		accountType = service.AccountTypeOAuth
	}
	if !supplierOAuthKindSupported(platform, accountType) || (platform != service.PlatformGemini && (request.OAuthType != "" || request.ProjectID != "" || request.TierID != "")) {
		response.ErrorFrom(c, service.ErrSupplierAccountKindInvalid)
		return
	}
	if platform == service.PlatformGemini && request.OAuthType != "" && request.OAuthType != "code_assist" && request.OAuthType != "google_one" && request.OAuthType != "ai_studio" {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrSupplierTokenInvalid)
		return
	}
	if err := h.accounts.AuthorizeOAuth(c.Request.Context(), supplierID, platform, accountType); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	proxies, err := h.accounts.AvailableProxies(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	validProxy := false
	for _, proxy := range proxies {
		if proxy.ID == request.ProxyID {
			validProxy = true
			break
		}
	}
	if !validProxy {
		response.ErrorFrom(c, service.ErrSupplierProxyRequired)
		return
	}
	var result any
	switch platform {
	case service.PlatformOpenAI:
		result, err = h.openAIOAuth.GenerateSupplierAuthURL(c.Request.Context(), supplierID, request.ProxyID)
	case service.PlatformAnthropic:
		result, err = h.claudeOAuth.GenerateSupplierAuthURL(c.Request.Context(), supplierID, request.ProxyID, accountType == service.AccountTypeSetupToken)
	case service.PlatformGemini:
		oauthType := request.OAuthType
		if oauthType == "" {
			oauthType = "code_assist"
		}
		result, err = h.geminiOAuth.GenerateSupplierAuthURL(c.Request.Context(), supplierID, request.ProxyID, request.ProjectID, oauthType, request.TierID)
	case service.PlatformAntigravity:
		result, err = h.antigravityOAuth.GenerateSupplierAuthURL(c.Request.Context(), supplierID, request.ProxyID)
	case service.PlatformGrok:
		result, err = h.grokOAuth.GenerateSupplierAuthURL(c.Request.Context(), supplierID, request.ProxyID)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID})
	response.Success(c, result)
}

func (h *SupplierHandler) ExchangeSupplierOAuthCode(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request struct {
		SessionID string `json:"session_id"`
		Code      string `json:"code"`
		State     string `json:"state"`
		Type      string `json:"type"`
	}
	if err := decodeStrictSupplierJSON(c, &request); err != nil || strings.TrimSpace(request.SessionID) == "" || strings.TrimSpace(request.Code) == "" {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	platform := c.Param("platform")
	accountType := request.Type
	if accountType == "" {
		accountType = service.AccountTypeOAuth
	}
	if !supplierOAuthKindSupported(platform, accountType) {
		response.ErrorFrom(c, service.ErrSupplierAccountKindInvalid)
		return
	}
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrSupplierTokenInvalid)
		return
	}
	if err := h.accounts.AuthorizeOAuth(c.Request.Context(), supplierID, platform, accountType); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var result any
	var err error
	switch platform {
	case service.PlatformOpenAI:
		result, err = h.openAIOAuth.ExchangeSupplierCode(c.Request.Context(), supplierID, &service.OpenAIExchangeCodeInput{SessionID: request.SessionID, Code: request.Code, State: request.State})
	case service.PlatformAnthropic:
		result, err = h.claudeOAuth.ExchangeSupplierCode(c.Request.Context(), supplierID, accountType == service.AccountTypeSetupToken, &service.ExchangeCodeInput{SessionID: request.SessionID, Code: request.Code})
	case service.PlatformGemini:
		result, err = h.geminiOAuth.ExchangeSupplierCode(c.Request.Context(), supplierID, &service.GeminiExchangeCodeInput{SessionID: request.SessionID, Code: request.Code, State: request.State})
	case service.PlatformAntigravity:
		result, err = h.antigravityOAuth.ExchangeSupplierCode(c.Request.Context(), supplierID, &service.AntigravityExchangeCodeInput{SessionID: request.SessionID, Code: request.Code, State: request.State})
	case service.PlatformGrok:
		result, err = h.grokOAuth.ExchangeSupplierCode(c.Request.Context(), supplierID, &service.GrokExchangeCodeInput{SessionID: request.SessionID, Code: request.Code, State: request.State})
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID})
	response.Success(c, result)
}

func supplierOAuthKindSupported(platform, accountType string) bool {
	if accountType == service.AccountTypeSetupToken {
		return platform == service.PlatformAnthropic
	}
	if accountType != service.AccountTypeOAuth {
		return false
	}
	switch platform {
	case service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGemini, service.PlatformAntigravity, service.PlatformGrok:
		return true
	default:
		return false
	}
}

func (h *SupplierHandler) GetGrokOAuthCapabilities(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	response.Success(c, h.grokOAuth.GetCapabilities())
}

func (h *SupplierHandler) ExchangeSupplierOAuthCredential(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request struct {
		ProxyID    int64  `json:"proxy_id"`
		Type       string `json:"type"`
		Method     string `json:"method"`
		SessionKey string `json:"session_key"`
		SSOToken   string `json:"sso_token"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := decodeStrictSupplierJSON(c, &request); err != nil || request.ProxyID <= 0 {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	platform := c.Param("platform")
	accountType := request.Type
	if accountType == "" {
		accountType = service.AccountTypeOAuth
	}
	if !supplierOAuthKindSupported(platform, accountType) ||
		!(platform == service.PlatformAnthropic && request.Method == "cookie" && request.SessionKey != "" && request.SSOToken == "" && request.Email == "" && request.Password == "" ||
			platform == service.PlatformGrok && accountType == service.AccountTypeOAuth && request.Method == "sso" && request.SSOToken != "" && request.SessionKey == "" && request.Email == "" && request.Password == "" ||
			platform == service.PlatformGrok && accountType == service.AccountTypeOAuth && request.Method == "password" && request.Email != "" && request.Password != "" && request.SessionKey == "" && request.SSOToken == "") {
		response.ErrorFrom(c, service.ErrSupplierAccountInputInvalid)
		return
	}
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrSupplierTokenInvalid)
		return
	}
	if err := h.accounts.AuthorizeOAuth(c.Request.Context(), supplierID, platform, accountType); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	proxies, err := h.accounts.AvailableProxies(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	validProxy := false
	for _, proxy := range proxies {
		if proxy.ID == request.ProxyID {
			validProxy = true
			break
		}
	}
	if !validProxy {
		response.ErrorFrom(c, service.ErrSupplierProxyRequired)
		return
	}
	var result any
	proxyID := request.ProxyID
	switch request.Method {
	case "cookie":
		scope := "full"
		if accountType == service.AccountTypeSetupToken {
			scope = "inference"
		}
		result, err = h.claudeOAuth.CookieAuth(c.Request.Context(), &service.CookieAuthInput{SessionKey: request.SessionKey, ProxyID: &proxyID, Scope: scope})
	case "sso":
		result, err = h.grokOAuth.ValidateSSOToken(c.Request.Context(), request.SSOToken, &proxyID)
	case "password":
		result, err = h.grokOAuth.AuthorizePassword(c.Request.Context(), request.Email, request.Password, &proxyID)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID})
	response.Success(c, result)
}
