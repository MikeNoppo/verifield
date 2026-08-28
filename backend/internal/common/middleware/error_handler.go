package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/response"
	"verifield-be/internal/common/validation"
	"verifield-be/internal/shared/ctxkey"
)

// ErrorHandler adalah padanan ExceptionFilter di NestJS.
//
// Controller tidak menulis response error sendiri; cukup panggil
// c.Error(err) lalu return. Middleware ini berjalan setelah c.Next(),
// membaca error terakhir, dan menerjemahkannya jadi response yang seragam.
func ErrorHandler(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		status, code, message, fields := classify(err)

		attrs := []any{
			"status", status,
			"code", code,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"request_id", ctxkey.RequestID(c),
			"error", err.Error(),
		}
		if status >= http.StatusInternalServerError {
			log.Error("request gagal", attrs...)
		} else {
			log.Warn("request ditolak", attrs...)
		}

		// Kalau handler sudah sempat menulis body, jangan tulis dua kali.
		if c.Writer.Written() {
			return
		}

		response.Error(c, status, code, message, fields)
	}
}

// classify memetakan error apa pun menjadi (status, code, message, fields).
// Pesan error internal tidak pernah dibocorkan ke client — hanya masuk log.
func classify(err error) (int, string, string, []apperror.FieldError) {
	if appErr, ok := apperror.As(err); ok {
		return appErr.Status, appErr.Code, appErr.Message, appErr.Fields
	}

	if fields := validation.Translate(err); len(fields) > 0 {
		return http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validasi gagal", fields
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, "NOT_FOUND", "Data tidak ditemukan", nil
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return http.StatusConflict, "CONFLICT", "Data sudah ada", nil
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return http.StatusConflict, "CONFLICT", "Data masih terhubung dengan data lain", nil
	}

	return http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server", nil
}
