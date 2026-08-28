// Package ctxkey memusatkan key yang disimpan di gin.Context, supaya tidak ada
// string ajaib yang tersebar di middleware dan controller.
package ctxkey

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/schema"
)

const (
	requestID = "request.id"
	actor     = "actor.user"
)

// SetActor menyimpan pengguna yang sedang melakukan aksi.
func SetActor(c *gin.Context, user *schema.User) {
	c.Set(actor, user)
}

// Actor mengambil pengguna yang sedang melakukan aksi.
func Actor(c *gin.Context) (*schema.User, bool) {
	raw, exists := c.Get(actor)
	if !exists {
		return nil, false
	}
	user, ok := raw.(*schema.User)
	return user, ok && user != nil
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
