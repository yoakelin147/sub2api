package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type supplierTokenAuthenticator interface {
	Authenticate(context.Context, string) (*service.Supplier, error)
}

type supplierReader interface {
	GetByID(context.Context, int64) (*service.Supplier, error)
}

func NewSupplierAuthMiddleware(
	tokens *service.SupplierTokenService,
	suppliers *service.SupplierService,
) SupplierAuthMiddleware {
	return SupplierAuthMiddleware(supplierAuth(tokens, suppliers))
}

func supplierAuth(tokens supplierTokenAuthenticator, suppliers supplierReader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if subject, ok := GetAuthSubjectFromContext(c); ok {
			role, _ := GetUserRoleFromContext(c)
			if role != service.RoleSupplier || subject.SupplierID == nil {
				AbortWithError(c, 403, "SUPPLIER_ACCESS_REQUIRED", "Supplier access required")
				return
			}
			supplier, err := suppliers.GetByID(c.Request.Context(), *subject.SupplierID)
			if err != nil || supplier.Status != domain.SupplierStatusActive {
				AbortWithError(c, 403, "SUPPLIER_DISABLED", "Supplier is disabled")
				return
			}
			setSupplierContext(c, supplier.ID, service.AuditAuthMethodSupplierJWT)
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader("x-api-key"))
		if token == "" {
			AbortWithError(c, 401, "SUPPLIER_AUTH_REQUIRED", "Supplier authentication required")
			return
		}
		supplier, err := tokens.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, service.ErrSupplierDisabled) {
				AbortWithError(c, 403, "SUPPLIER_DISABLED", "Supplier is disabled")
				return
			}
			AbortWithError(c, 401, "INVALID_SUPPLIER_TOKEN", "Invalid supplier token")
			return
		}
		c.Set(string(ContextKeyUser), AuthSubject{SupplierID: &supplier.ID})
		c.Set(string(ContextKeyUserRole), service.RoleSupplier)
		c.Set(ContextKeyAuthEmail, "supplier:"+supplier.Code)
		setSupplierContext(c, supplier.ID, service.AuditAuthMethodSupplierAPIKey)
		c.Next()
	}
}

func setSupplierContext(c *gin.Context, supplierID int64, authMethod string) {
	c.Set(string(ContextKeySupplierID), supplierID)
	c.Set("auth_method", authMethod)
}

// SupplierJWTOnly prevents a machine token from changing or revoking itself.
func SupplierJWTOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("auth_method") != service.AuditAuthMethodSupplierJWT {
			AbortWithError(c, 403, "SUPPLIER_JWT_REQUIRED", "Supplier member session required")
			return
		}
		c.Next()
	}
}
