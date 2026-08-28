// Package database mengelola koneksi GORM ke PostgreSQL beserta migrasinya.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"verifield-be/internal/common/config"
)

// Connect membuka koneksi ke PostgreSQL, menyetel connection pool, dan
// memastikan database benar-benar bisa dihubungi (ping) sebelum aplikasi jalan.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:         newGormLogger(cfg.LogLevel),
		NowFunc:        func() time.Time { return time.Now().UTC() },
		TranslateError: true, // ubah error driver jadi gorm.ErrDuplicatedKey dkk
	})
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mengambil *sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// Close menutup connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func newGormLogger(level string) gormlogger.Interface {
	var logLevel gormlogger.LogLevel
	switch level {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "info":
		logLevel = gormlogger.Info
	default:
		logLevel = gormlogger.Warn
	}

	return gormlogger.New(
		slogWriter{},
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

// slogWriter menyalurkan log GORM ke slog supaya formatnya seragam dengan
// log aplikasi lainnya.
type slogWriter struct{}

func (slogWriter) Printf(format string, args ...any) {
	slog.Default().Debug(fmt.Sprintf(format, args...), "source", "gorm")
}
