// Package ctxkey memusatkan key yang disimpan di gin.Context, supaya tidak ada
// string ajaib yang tersebar di middleware dan controller.
package ctxkey

import "github.com/gin-gonic/gin"

const (
	userID    = "auth.user_id"
	userRole  = "auth.user_role"
	requestID = "request.id"
)

// SetUser menyimpan identitas hasil verifikasi JWT ke context.
func SetUser(c *gin.Context, id, role string) {
	c.Set(userID, id)
	c.Set(userRole, role)
}

// UserID mengambil id user yang sedang login.
func UserID(c *gin.Context) (string, bool) {
	return stringValue(c, userID)
}

// UserRole mengambil role user yang sedang login.
func UserRole(c *gin.Context) (string, bool) {
	return stringValue(c, userRole)
}

// SetRequestID menyimpan id request untuk keperluan tracing di log.
func SetRequestID(c *gin.Context, id string) {
	c.Set(requestID, id)
}

// RequestID mengambil id request; mengembalikan string kosong kalau belum diset.
func RequestID(c *gin.Context) string {
	value, _ := stringValue(c, requestID)
	return value
}

func stringValue(c *gin.Context, key string) (string, bool) {
	raw, exists := c.Get(key)
	if !exists {
		return "", false
	}
	value, ok := raw.(string)
	return value, ok && value != ""
}
