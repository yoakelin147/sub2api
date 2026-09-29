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
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	supplier := v1.Group("/supplier")
	supplier.Use(gin.HandlerFunc(optionalJWT))
	supplier.Use(gin.HandlerFunc(supplierAuth))
	supplier.Use(panelRateLimiter.Supplier())
	supplier.Use(gin.HandlerFunc(auditLog))
	supplier.GET("/me", h.Me)
	supplier.GET("/proxies", h.ListProxies)
	supplier.GET("/oauth/grok/capabilities", h.GetGrokOAuthCapabilities)
	supplier.POST("/oauth/:platform/auth-url", h.GenerateSupplierOAuthURL)
	supplier.POST("/oauth/:platform/exchange-code", h.ExchangeSupplierOAuthCode)
	supplier.POST("/oauth/:platform/credential-exchange", h.ExchangeSupplierOAuthCredential)
	accounts := supplier.Group("/accounts")
	accounts.GET("", h.ListAccounts)
	accounts.POST("", h.CreateAccount)
	accounts.POST("/batch", h.BatchCreateAccounts)
	accounts.GET("/:id", h.GetAccount)
	accounts.GET("/:id/models", h.GetAccountTestModels)
	accounts.PUT("/:id", h.UpdateAccount)
	accounts.DELETE("/:id", h.DeleteAccount)
	accounts.POST("/:id/test", panelRateLimiter.SupplierTest(), h.TestAccount)

	accessToken := supplier.Group("/access-token")
	accessToken.Use(middleware.SupplierJWTOnly())
	accessToken.GET("", h.AccessTokenStatus)
	accessToken.POST("/regenerate", h.RegenerateAccessToken)
	accessToken.DELETE("", h.RevokeAccessToken)
}
