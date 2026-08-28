// Package apperror adalah padanan HttpException di NestJS: error yang membawa
// status HTTP, kode mesin, dan pesan yang aman ditampilkan ke client.
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// FieldError adalah detail kesalahan per field, dipakai untuk hasil validasi DTO.
type FieldError struct {
	Field   string `json:"field"   example:"email"`
	Message string `json:"message" example:"email wajib diisi"`
}

// AppError adalah error yang sudah "tahu" harus jadi response HTTP seperti apa.
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  []FieldError
	Err     error // penyebab asli, hanya untuk log — tidak pernah dikirim ke client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// Wrap menempelkan error penyebab tanpa mengubah pesan yang dilihat client.
func (e *AppError) Wrap(err error) *AppError {
	clone := *e
	clone.Err = err
	return &clone
}

// WithFields menempelkan detail error per field (hasil validasi).
func (e *AppError) WithFields(fields []FieldError) *AppError {
	clone := *e
	clone.Fields = fields
	return &clone
}

// New membuat AppError dengan status dan kode kustom.
func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, "BAD_REQUEST", message)
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(message string) *AppError {
	return New(http.StatusConflict, "CONFLICT", message)
}

func UnprocessableEntity(message string) *AppError {
	return New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

func Internal(message string) *AppError {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// As mengekstrak *AppError dari rantai error, kalau ada.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
