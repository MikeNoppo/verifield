package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/shared/ctxkey"
	"verifield-be/internal/shared/jwtx"
)

// JWTAuth adalah padanan JwtAuthGuard di NestJS: memverifikasi header
// `Authorization: Bearer <token>` lalu menaruh identitas user ke context.
func JWTAuth(manager *jwtx.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := bearerToken(c)
		if err != nil {
			c.Error(err) //nolint:errcheck // ditangani ErrorHandler
			c.Abort()
			return
		}

		claims, err := manager.Parse(token, jwtx.TypeAccess)
		if err != nil {
			c.Error(authError(err)) //nolint:errcheck // ditangani ErrorHandler
			c.Abort()
			return
		}

		ctxkey.SetUser(c, claims.Subject, claims.Role)
		c.Next()
	}
}

// OptionalJWTAuth mengisi identitas user kalau token valid, tetapi tetap
// meneruskan request kalau token tidak ada. Berguna untuk endpoint publik yang
// menampilkan data ekstra bagi user yang login.
func OptionalJWTAuth(manager *jwtx.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := bearerToken(c); err == nil {
			if claims, err := manager.Parse(token, jwtx.TypeAccess); err == nil {
				ctxkey.SetUser(c, claims.Subject, claims.Role)
			}
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) (string, error) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return "", apperror.Unauthorized("Header Authorization tidak ditemukan")
	}

	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", apperror.Unauthorized("Format Authorization harus 'Bearer <token>'")
	}

	return strings.TrimSpace(token), nil
}

func authError(err error) error {
	switch {
	case errors.Is(err, jwtx.ErrExpiredToken):
		return apperror.Unauthorized("Token sudah kedaluwarsa").Wrap(err)
	case errors.Is(err, jwtx.ErrWrongType):
		return apperror.Unauthorized("Jenis token tidak sesuai").Wrap(err)
	default:
		return apperror.Unauthorized("Token tidak valid").Wrap(err)
	}
}
