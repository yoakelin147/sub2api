package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	suppliers *service.SupplierService
	tokens    *service.SupplierTokenService
	accounts  *service.SupplierAccountService
	tester    *service.AccountTestService
}

func NewSupplierHandler(
	suppliers *service.SupplierService,
	tokens *service.SupplierTokenService,
	accounts *service.SupplierAccountService,
	tester *service.AccountTestService,
) *SupplierHandler {
	return &SupplierHandler{suppliers: suppliers, tokens: tokens, accounts: accounts, tester: tester}
}

func (h *SupplierHandler) Me(c *gin.Context) {
	supplier := h.currentSupplier(c)
	if supplier == nil {
		return
	}
	response.Success(c, supplierResponse(supplier))
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
	return gin.H{
		"id":                    supplier.ID,
		"code":                  supplier.Code,
		"name":                  supplier.Name,
		"status":                supplier.Status,
		"allowed_account_kinds": supplier.AllowedAccountKinds,
		"access_token":          tokenStatusResponse(supplier),
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
