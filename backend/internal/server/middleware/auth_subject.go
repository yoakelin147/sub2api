package middleware

import "github.com/gin-gonic/gin"

// AuthSubject is the minimal authenticated identity stored in gin context.
// Decision: {UserID int64, Concurrency int}
type AuthSubject struct {
	UserID      int64
	Concurrency int
	SupplierID  *int64
}

func GetAuthSubjectFromContext(c *gin.Context) (AuthSubject, bool) {
	value, exists := c.Get(string(ContextKeyUser))
	if !exists {
		return AuthSubject{}, false
	}
	subject, ok := value.(AuthSubject)
	return subject, ok
}

func GetSupplierIDFromContext(c *gin.Context) (int64, bool) {
	value, exists := c.Get(string(ContextKeySupplierID))
	if !exists {
		return 0, false
	}
	supplierID, ok := value.(int64)
	return supplierID, ok && supplierID > 0
}

func GetUserRoleFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}
