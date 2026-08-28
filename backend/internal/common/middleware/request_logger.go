package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"verifield-be/internal/shared/ctxkey"
)

// HeaderRequestID adalah header yang dipakai untuk korelasi antar service.
const HeaderRequestID = "X-Request-ID"

// RequestID memakai ulang X-Request-ID dari client kalau ada, atau membuat yang
// baru, lalu mengembalikannya lewat response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}

		ctxkey.SetRequestID(c, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// RequestLogger mencatat setiap request beserta durasi dan status akhirnya.
func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"request_id", ctxkey.RequestID(c),
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		log.Info("http request", attrs...)
	}
}
