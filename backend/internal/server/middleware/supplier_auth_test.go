package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupplierAuthAcceptsSupplierJWTContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	supplierID := int64(7)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 9, SupplierID: &supplierID})
		c.Set(string(ContextKeyUserRole), service.RoleSupplier)
		c.Next()
	})
	router.Use(supplierAuth(nil, supplierReaderStub{supplier: activeSupplier(supplierID)}))
	router.GET("/test", func(c *gin.Context) {
		id, ok := GetSupplierIDFromContext(c)
		require.True(t, ok)
		require.Equal(t, supplierID, id)
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestSupplierAuthAcceptsSupplierTokenAndRejectsOtherRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	supplier := activeSupplier(8)
	router := gin.New()
	router.Use(supplierAuth(supplierTokenAuthenticatorStub{supplier: supplier}, supplierReaderStub{}))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("x-api-key", "supplier_test")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	userRouter := gin.New()
	userRouter.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 3})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	userRouter.Use(supplierAuth(nil, supplierReaderStub{}))
	userRouter.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder = httptest.NewRecorder()
	userRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestSupplierJWTOnlyRejectsMachineToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name       string
		authMethod string
		want       int
	}{
		{name: "member jwt", authMethod: service.AuditAuthMethodSupplierJWT, want: http.StatusOK},
		{name: "machine token", authMethod: service.AuditAuthMethodSupplierAPIKey, want: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("auth_method", tt.authMethod)
				c.Next()
			})
			router.Use(SupplierJWTOnly())
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
			require.Equal(t, tt.want, recorder.Code)
		})
	}
}

type supplierTokenAuthenticatorStub struct {
	supplier *service.Supplier
	err      error
}

func (s supplierTokenAuthenticatorStub) Authenticate(context.Context, string) (*service.Supplier, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.supplier == nil {
		return nil, errors.New("missing supplier")
	}
	return s.supplier, nil
}

type supplierReaderStub struct {
	supplier *service.Supplier
}

func (s supplierReaderStub) GetByID(context.Context, int64) (*service.Supplier, error) {
	if s.supplier == nil {
		return nil, service.ErrSupplierNotFound
	}
	return s.supplier, nil
}

func activeSupplier(id int64) *service.Supplier {
	return &service.Supplier{ID: id, Code: "demo", Status: domain.SupplierStatusActive}
}
