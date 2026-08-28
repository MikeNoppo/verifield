package middleware

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/shared/ctxkey"
)

// RequireRoles adalah padanan RolesGuard + @Roles() di NestJS.
// Wajib dipasang SETELAH JWTAuth pada rantai handler.
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role, ok := ctxkey.UserRole(c)
		if !ok {
			c.Error(apperror.Unauthorized("Autentikasi diperlukan")) //nolint:errcheck
			c.Abort()
			return
		}

		if _, permitted := allowed[role]; !permitted {
			c.Error(apperror.Forbidden("Anda tidak punya akses ke resource ini")) //nolint:errcheck
			c.Abort()
			return
		}

		c.Next()
	}
}
