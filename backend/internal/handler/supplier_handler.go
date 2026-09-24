package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	suppliers        *service.SupplierService
	tokens           *service.SupplierTokenService
	accounts         *service.SupplierAccountService
	tester           *service.AccountTestService
	openAIOAuth      *service.OpenAIOAuthService
	claudeOAuth      *service.OAuthService
	geminiOAuth      *service.GeminiOAuthService
	antigravityOAuth *service.AntigravityOAuthService
	grokOAuth        *service.GrokOAuthService
}

func NewSupplierHandler(
	suppliers *service.SupplierService,
	tokens *service.SupplierTokenService,
	accounts *service.SupplierAccountService,
	tester *service.AccountTestService,
	openAIOAuth *service.OpenAIOAuthService,
	claudeOAuth *service.OAuthService,
	geminiOAuth *service.GeminiOAuthService,
	antigravityOAuth *service.AntigravityOAuthService,
	grokOAuth *service.GrokOAuthService,
) *SupplierHandler {
	return &SupplierHandler{suppliers: suppliers, tokens: tokens, accounts: accounts, tester: tester, openAIOAuth: openAIOAuth, claudeOAuth: claudeOAuth, geminiOAuth: geminiOAuth, antigravityOAuth: antigravityOAuth, grokOAuth: grokOAuth}
}

func (h *SupplierHandler) Me(c *gin.Context) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	supplier, err := h.suppliers.GetWithStats(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groups, err := h.accounts.AuthorizedGroups(c.Request.Context(), supplier)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	profile := supplierResponse(supplier)
	profile["authorized_groups"] = groups
	response.Success(c, profile)
}

func (h *SupplierHandler) AccessTokenStatus(c *gin.Context) {
	supplier := h.currentSupplier(c)
	if supplier == nil {
		return
	}
	response.Success(c, tokenStatusResponse(supplier))
}

func (h *SupplierHandler) RegenerateAccessToken(c *gin.Context) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	token, err := h.tokens.Regenerate(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID})
	response.Success(c, gin.H{"key": token})
}

func (h *SupplierHandler) RevokeAccessToken(c *gin.Context) {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return
	}
	if err := h.tokens.Revoke(c.Request.Context(), supplierID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{"supplier_id": supplierID})
	response.Success(c, gin.H{"message": "Supplier access token revoked"})
}

func (h *SupplierHandler) currentSupplier(c *gin.Context) *service.Supplier {
	supplierID, ok := middleware.GetSupplierIDFromContext(c)
	if !ok {
		middleware.AbortWithError(c, http.StatusUnauthorized, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
		return nil
	}
	supplier, err := h.suppliers.GetByID(c.Request.Context(), supplierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return nil
	}
	return supplier
}

func supplierResponse(supplier *service.Supplier) gin.H {
	approvedGroups := map[string][]int64{}
	if !supplier.ReviewRequired {
		approvedGroups = supplier.AutoApproveGroups
	}
	return gin.H{
		"id":                    supplier.ID,
		"code":                  supplier.Code,
		"name":                  supplier.Name,
		"status":                supplier.Status,
		"allowed_account_kinds": append([]service.SupplierAccountKind{}, supplier.AllowedAccountKinds...),
		"review_required":       supplier.ReviewRequired,
		"auto_approve_groups":   approvedGroups,
		"access_token":          tokenStatusResponse(supplier),
		"stats":                 supplier.Stats,
	}
}

func tokenStatusResponse(supplier *service.Supplier) gin.H {
	return gin.H{
		"exists":       supplier.TokenHash != nil,
		"masked_key":   supplier.TokenPrefix,
		"created_at":   supplier.TokenCreatedAt,
		"last_used_at": supplier.TokenLastUsedAt,
	}
}
