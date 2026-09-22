package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAllowRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		role string
		want int
	}{
		{name: "user", role: service.RoleUser, want: http.StatusOK},
		{name: "admin", role: service.RoleAdmin, want: http.StatusOK},
		{name: "supplier", role: service.RoleSupplier, want: http.StatusForbidden},
		{name: "missing", want: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			if tt.role != "" {
				router.Use(func(c *gin.Context) {
					c.Set(string(ContextKeyUserRole), tt.role)
					c.Next()
				})
			}
			router.Use(AllowRoles(service.RoleAdmin, service.RoleUser))
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
			require.Equal(t, tt.want, recorder.Code)
		})
	}
}
