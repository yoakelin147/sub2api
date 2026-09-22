package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterSupplierRoutes(
	v1 *gin.RouterGroup,
	h *handler.SupplierHandler,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	supplierAuth middleware.SupplierAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
) {
	supplier := v1.Group("/supplier")
	supplier.Use(gin.HandlerFunc(optionalJWT))
	supplier.Use(gin.HandlerFunc(supplierAuth))
	supplier.Use(gin.HandlerFunc(auditLog))
	supplier.GET("/me", h.Me)
	accounts := supplier.Group("/accounts")
	accounts.GET("", h.ListAccounts)
	accounts.POST("", h.CreateAccount)
	accounts.POST("/batch", h.BatchCreateAccounts)
	accounts.GET("/:id", h.GetAccount)
	accounts.PUT("/:id", h.UpdateAccount)
	accounts.DELETE("/:id", h.DeleteAccount)
	accounts.POST("/:id/test", h.TestAccount)

	accessToken := supplier.Group("/access-token")
	accessToken.Use(middleware.SupplierJWTOnly())
	accessToken.GET("", h.AccessTokenStatus)
	accessToken.POST("/regenerate", h.RegenerateAccessToken)
	accessToken.DELETE("", h.RevokeAccessToken)
}
