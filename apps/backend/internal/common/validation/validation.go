// Package validation menerjemahkan error dari go-playground/validator menjadi
// pesan per field yang terbaca manusia. Padanan ValidationPipe di NestJS.
package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"verifield-be/internal/common/apperror"
)

// Init mendaftarkan tag name function supaya nama field pada pesan error
// memakai nama JSON (`email`), bukan nama field Go (`Email`).
// Dipanggil sekali saat bootstrap aplikasi.
func Init() {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			name := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})
}

// Translate mengubah validator.ValidationErrors menjadi []FieldError.
// Mengembalikan nil kalau err bukan error validasi.
func Translate(err error) []apperror.FieldError {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return nil
	}

	fields := make([]apperror.FieldError, 0, len(validationErrs))
	for _, fieldErr := range validationErrs {
		fields = append(fields, apperror.FieldError{
			Field:   fieldErr.Field(),
			Message: messageFor(fieldErr),
		})
	}
	return fields
}

func messageFor(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "email":
		return fmt.Sprintf("%s harus berupa alamat email yang valid", field)
	case "url":
		return fmt.Sprintf("%s harus berupa URL yang valid", field)
	case "uuid", "uuid4":
		return fmt.Sprintf("%s harus berupa UUID yang valid", field)
	case "min":
		return fmt.Sprintf("%s minimal %s%s", field, fe.Param(), unitFor(fe))
	case "max":
		return fmt.Sprintf("%s maksimal %s%s", field, fe.Param(), unitFor(fe))
	case "len":
		return fmt.Sprintf("%s harus tepat %s%s", field, fe.Param(), unitFor(fe))
	case "gt":
		return fmt.Sprintf("%s harus lebih besar dari %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s harus lebih besar atau sama dengan %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s harus lebih kecil dari %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s harus lebih kecil atau sama dengan %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari: %s", field, strings.ReplaceAll(fe.Param(), " ", ", "))
	case "eqfield":
		return fmt.Sprintf("%s harus sama dengan %s", field, fe.Param())
	case "numeric":
		return fmt.Sprintf("%s harus berupa angka", field)
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh berisi huruf dan angka", field)
	case "boolean":
		return fmt.Sprintf("%s harus bernilai true atau false", field)
	default:
		return fmt.Sprintf("%s tidak memenuhi aturan %s", field, fe.Tag())
	}
}

// unitFor memberi satuan yang tepat: "karakter" untuk string, "item" untuk
// slice/map, dan tanpa satuan untuk angka.
func unitFor(fe validator.FieldError) string {
	switch fe.Kind() {
	case reflect.String:
		return " karakter"
	case reflect.Slice, reflect.Array, reflect.Map:
		return " item"
	default:
		return ""
	}
}
