// Package config memuat seluruh konfigurasi aplikasi dari file .env dan
// environment variable. Padanan ConfigModule (@nestjs/config) di NestJS.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config adalah root dari seluruh konfigurasi aplikasi.
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Seed     SeedConfig
}

type AppConfig struct {
	Name  string
	Env   string
	Debug bool

	// TimeZone adalah zona waktu operasi lapangan (asumsi A-04: satu zona).
	// Dipakai aturan jam kerja pada penjadwalan, bukan untuk penyimpanan —
	// seluruh waktu tetap disimpan dalam UTC.
	TimeZone string
}

func (a AppConfig) IsProduction() bool { return a.Env == "production" }

// Location memuat zona waktu operasi.
//
// WARNING: image runtime wajib memuat tzdata. Alpine tanpa paket itu membuat
// LoadLocation gagal untuk zona apa pun selain UTC, dan aturan jam kerja akan
// dihitung terhadap zona yang keliru.
func (a AppConfig) Location() (*time.Location, error) {
	if a.TimeZone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(a.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("APP_TIMEZONE %q tidak dikenali: %w", a.TimeZone, err)
	}
	return loc, nil
}

type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

func (h HTTPConfig) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	TimeZone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string
}

// DSN membentuk connection string untuk driver PostgreSQL.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone,
	)
}

// SeedConfig adalah kredensial admin pertama, dipakai oleh cmd/seeder.
type SeedConfig struct {
	AdminName     string
	AdminEmail    string
	AdminPassword string
}

// Load membaca .env (kalau ada) lalu menimpanya dengan environment variable.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		// .env bersifat opsional: di production konfigurasi biasanya
		// disuntikkan lewat environment variable saja.
		var notFound viper.ConfigFileNotFoundError
		var pathErr *fs.PathError
		if !errors.As(err, &notFound) && !errors.As(err, &pathErr) {
			return nil, fmt.Errorf("membaca .env: %w", err)
		}
	}

	cfg := &Config{
		App: AppConfig{
			Name:     v.GetString("APP_NAME"),
			Env:      v.GetString("APP_ENV"),
			Debug:    v.GetBool("APP_DEBUG"),
			TimeZone: v.GetString("APP_TIMEZONE"),
		},
		HTTP: HTTPConfig{
			Host:            v.GetString("HTTP_HOST"),
			Port:            v.GetInt("HTTP_PORT"),
			ReadTimeout:     v.GetDuration("HTTP_READ_TIMEOUT"),
			WriteTimeout:    v.GetDuration("HTTP_WRITE_TIMEOUT"),
			ShutdownTimeout: v.GetDuration("HTTP_SHUTDOWN_TIMEOUT"),
			AllowedOrigins:  splitAndTrim(v.GetString("HTTP_ALLOWED_ORIGINS")),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetInt("DB_PORT"),
			User:            v.GetString("DB_USER"),
			Password:        v.GetString("DB_PASSWORD"),
			Name:            v.GetString("DB_NAME"),
			SSLMode:         v.GetString("DB_SSLMODE"),
			TimeZone:        v.GetString("DB_TIMEZONE"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetDuration("DB_CONN_MAX_LIFETIME"),
			LogLevel:        v.GetString("DB_LOG_LEVEL"),
		},
		Seed: SeedConfig{
			AdminName:     v.GetString("SEED_ADMIN_NAME"),
			AdminEmail:    v.GetString("SEED_ADMIN_EMAIL"),
			AdminPassword: v.GetString("SEED_ADMIN_PASSWORD"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_NAME", "verifield-be")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_DEBUG", true)
	v.SetDefault("APP_TIMEZONE", "Asia/Jakarta")

	v.SetDefault("HTTP_HOST", "0.0.0.0")
	v.SetDefault("HTTP_PORT", 8080)
	v.SetDefault("HTTP_READ_TIMEOUT", "15s")
	v.SetDefault("HTTP_WRITE_TIMEOUT", "15s")
	v.SetDefault("HTTP_SHUTDOWN_TIMEOUT", "10s")
	v.SetDefault("HTTP_ALLOWED_ORIGINS", "*")

	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "verifield")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("DB_TIMEZONE", "Asia/Jakarta")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 5)
	v.SetDefault("DB_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("DB_LOG_LEVEL", "warn")

	v.SetDefault("SEED_ADMIN_NAME", "Administrator")
	v.SetDefault("SEED_ADMIN_EMAIL", "admin@verifield.id")
}

// validate menangkap salah konfigurasi sedini mungkin (fail fast saat boot).
func (c *Config) validate() error {
	if c.Database.Name == "" {
		return errors.New("DB_NAME wajib diisi")
	}
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("HTTP_PORT tidak valid: %d", c.HTTP.Port)
	}
	if _, err := c.App.Location(); err != nil {
		return err
	}
	return nil
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
