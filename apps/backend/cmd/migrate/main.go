// Command migrate menerapkan berkas migrasi ke database memakai goose.
//
// Berkas migrasi sudah ter-embed di dalam binary ini, jadi di server cukup ada
// satu berkas biner — tidak perlu folder migrations/, tidak perlu Atlas, tidak
// perlu Go toolchain.
//
// Padanan `prisma migrate deploy`:
//
//	./bin/migrate up
//
// Perintah lain: up-by-one, down, down-to <versi>, redo, status, version.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pressly/goose/v3"

	"verifield-be/internal/common/config"
	"verifield-be/internal/common/database"
	"verifield-be/migrations"
)

const usage = `migrate — penerap migrasi database verifield-be

Penggunaan:
  migrate <perintah> [argumen]

Perintah:
  up                 terapkan semua migrasi yang tertunda (~ prisma migrate deploy)
  up-by-one          terapkan satu migrasi berikutnya saja
  down               batalkan satu migrasi terakhir
  down-to <versi>    batalkan sampai versi tertentu (0 = kosongkan semua)
  redo               ulangi migrasi terakhir (down lalu up)
  status             tampilkan migrasi yang sudah & belum diterapkan
  version            tampilkan versi migrasi database saat ini

Konfigurasi database dibaca dari .env / environment variable, sama seperti aplikasi.
`

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(usage)
		return nil
	}

	command, rest := args[0], args[1:]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("memuat konfigurasi: %w", err)
	}

	// Memakai ulang koneksi yang sama dengan aplikasi supaya DSN hanya
	// didefinisikan di satu tempat.
	gormDB, err := database.Connect(cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(gormDB) }()

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("mengambil *sql.DB: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("menyetel dialect: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := goose.RunContext(ctx, command, sqlDB, ".", rest...); err != nil {
		if errors.Is(err, goose.ErrNoNextVersion) {
			fmt.Println("tidak ada migrasi yang tertunda")
			return nil
		}
		return err
	}

	return nil
}
