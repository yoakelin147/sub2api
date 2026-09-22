package middleware

import "github.com/gin-gonic/gin"

// AllowRoles authorizes requests whose JWT-backed role is in the allowlist.
// It must run after an authentication middleware has populated the role.
func AllowRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		if _, ok := allowed[role]; !ok {
			AbortWithError(c, 403, "FORBIDDEN", "Role is not allowed")
			return
		}
		c.Next()
	}
}
