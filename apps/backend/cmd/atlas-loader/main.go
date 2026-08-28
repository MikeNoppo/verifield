// Command atlas-loader mencetak DDL PostgreSQL hasil pembacaan model GORM di
// internal/schema ke stdout. Program ini dipanggil oleh Atlas (lihat atlas.hcl)
// sebagai sumber "schema yang diinginkan" saat menghitung diff migrasi.
//
// Tidak butuh koneksi database — GORM hanya menyusun DDL secara offline.
// Coba manual: go run ./cmd/atlas-loader
package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"

	"verifield-be/internal/schema"
)

func main() {
	// schema.All() adalah daftar tabel yang sama dengan yang dipakai aplikasi,
	// jadi internal/schema/schema.go tetap satu-satunya sumber kebenaran.
	stmts, err := gormschema.New("postgres").Load(schema.All()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal memuat schema gorm: %v\n", err)
		os.Exit(1)
	}

	if _, err := io.WriteString(os.Stdout, stmts); err != nil {
		fmt.Fprintf(os.Stderr, "gagal menulis DDL: %v\n", err)
		os.Exit(1)
	}
}
