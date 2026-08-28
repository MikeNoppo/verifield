// Command seeder membuat akun admin pertama dari kredensial di .env.
// Padanan `prisma db seed`.
//
//	./bin/seeder      (atau: make seed)
//
// Idempoten: kalau email-nya sudah terdaftar, akun itu dinaikkan jadi admin dan
// diaktifkan kembali — aman dijalankan berulang kali.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/config"
	"verifield-be/internal/common/database"
	"verifield-be/internal/common/logger"
	"verifield-be/internal/modules/user"
	userdto "verifield-be/internal/modules/user/dto"
	"verifield-be/internal/schema"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seeder: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("memuat konfigurasi: %w", err)
	}

	if cfg.Seed.AdminEmail == "" || cfg.Seed.AdminPassword == "" {
		return errors.New("SEED_ADMIN_EMAIL dan SEED_ADMIN_PASSWORD wajib diisi di .env")
	}

	log := logger.New(cfg.App.Env, cfg.App.Debug)

	db, err := database.Connect(cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	// Memakai ulang modul user supaya aturan hashing password dan validasi
	// email duplikat persis sama dengan yang dipakai endpoint HTTP.
	users := user.NewModule(db).Service
	ctx := context.Background()

	created, err := users.CreateEntity(ctx, userdto.CreateUserDTO{
		Name:     cfg.Seed.AdminName,
		Email:    cfg.Seed.AdminEmail,
		Password: cfg.Seed.AdminPassword,
		Role:     string(schema.RoleAdmin),
	})

	switch {
	case err == nil:
		log.Info("admin pertama berhasil dibuat",
			"id", created.ID.String(),
			"email", created.Email,
			"role", string(created.Role),
		)
		return nil

	case isConflict(err):
		// Sudah ada: naikkan jadi admin supaya perintah ini tetap idempoten.
		return promoteExisting(ctx, users, cfg.Seed.AdminEmail, log)

	default:
		return err
	}
}

// promoteExisting menaikkan akun yang sudah ada menjadi admin aktif.
func promoteExisting(ctx context.Context, users user.Service, email string, log *slog.Logger) error {
	existing, err := users.FindEntityByEmail(ctx, email)
	if err != nil {
		return err
	}

	if existing.Role == schema.RoleAdmin && existing.IsActive {
		log.Info("admin sudah ada, tidak ada perubahan", "email", existing.Email)
		return nil
	}

	role := string(schema.RoleAdmin)
	active := true

	updated, err := users.Update(ctx, existing.ID.String(), userdto.UpdateUserDTO{
		Role:     &role,
		IsActive: &active,
	})
	if err != nil {
		return err
	}

	log.Info("akun yang sudah ada dinaikkan menjadi admin",
		"id", updated.ID,
		"email", updated.Email,
	)
	return nil
}

// isConflict mengenali error "email sudah terdaftar" dari user.Service.
func isConflict(err error) bool {
	appErr, ok := apperror.As(err)
	return ok && appErr.Code == "CONFLICT"
}
