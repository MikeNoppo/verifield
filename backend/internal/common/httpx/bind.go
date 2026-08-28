// Package httpx berisi helper tipis di atas Gin untuk binding request.
// Semua error binding sudah diubah jadi *apperror.AppError sehingga controller
// cukup memanggil c.Error(err) dan membiarkan ErrorHandler yang membentuk response.
package httpx

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/validation"
)

// BindJSON mem-parse body JSON ke dto sekaligus menjalankan validasinya.
func BindJSON(c *gin.Context, dto any) error {
	if err := c.ShouldBindJSON(dto); err != nil {
		return translate(err)
	}
	return nil
}

// BindQuery mem-parse query string ke dto sekaligus menjalankan validasinya.
func BindQuery(c *gin.Context, dto any) error {
	if err := c.ShouldBindQuery(dto); err != nil {
		return translate(err)
	}
	return nil
}

// BindURI mem-parse path parameter ke dto sekaligus menjalankan validasinya.
func BindURI(c *gin.Context, dto any) error {
	if err := c.ShouldBindUri(dto); err != nil {
		return translate(err)
	}
	return nil
}

func translate(err error) error {
	if fields := validation.Translate(err); len(fields) > 0 {
		return apperror.UnprocessableEntity("Validasi gagal").WithFields(fields)
	}

	if errors.Is(err, io.EOF) {
		return apperror.BadRequest("Request body tidak boleh kosong")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return apperror.BadRequest("Tipe data field '" + typeErr.Field + "' tidak sesuai").Wrap(err)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return apperror.BadRequest("Format JSON tidak valid").Wrap(err)
	}

	return apperror.BadRequest("Request tidak valid").Wrap(err)
}
