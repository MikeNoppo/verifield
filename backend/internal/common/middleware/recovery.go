package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/response"
	"verifield-be/internal/shared/ctxkey"
)

// Recovery menangkap panic, mencatat stack trace ke log, dan mengembalikan
// 500 yang bersih ke client tanpa membocorkan detail internal.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error("panic ter-recover",
			"panic", recovered,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"request_id", ctxkey.RequestID(c),
			"stack", string(debug.Stack()),
		)

		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Terjadi kesalahan pada server", nil)
	})
}
