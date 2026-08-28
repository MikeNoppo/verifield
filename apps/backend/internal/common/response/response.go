// Package response menyeragamkan bentuk JSON seluruh endpoint.
// Padanan TransformInterceptor di NestJS.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
)

// Meta adalah informasi paginasi untuk response berbentuk list.
type Meta struct {
	Page      int   `json:"page"       example:"1"`
	Limit     int   `json:"limit"      example:"10"`
	Total     int64 `json:"total"      example:"42"`
	TotalPage int   `json:"total_page" example:"5"`
}

// Envelope adalah bentuk baku seluruh response, sukses maupun gagal.
type Envelope struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Data    any                   `json:"data,omitempty"`
	Meta    *Meta                 `json:"meta,omitempty"`
	Code    string                `json:"code,omitempty"`
	Errors  []apperror.FieldError `json:"errors,omitempty"`
}

// OK mengirim 200 dengan data tunggal.
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data})
}

// Created mengirim 201.
func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

// Paginated mengirim 200 dengan data list beserta meta paginasi.
func Paginated(c *gin.Context, message string, data any, meta Meta) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data, Meta: &meta})
}

// NoContent mengirim 204 tanpa body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error mengirim response gagal dengan bentuk yang sama seperti response sukses.
func Error(c *gin.Context, status int, code, message string, fields []apperror.FieldError) {
	c.AbortWithStatusJSON(status, Envelope{
		Success: false,
		Message: message,
		Code:    code,
		Errors:  fields,
	})
}
