// Command api adalah entrypoint HTTP server. Padanan main.ts di NestJS.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"verifield-be/internal/app"
	"verifield-be/internal/common/config"
	"verifield-be/internal/common/logger"

	_ "verifield-be/docs" // registrasi spesifikasi swagger
)

// @title			Verifield API
// @version		1.0
// @description	REST API layanan inspeksi & sampling lapangan — Go + Gin dengan struktur modular ala NestJS.
// @description	Autentikasi di luar cakupan PoC: seluruh endpoint terbuka dan peran dipilih di frontend.
// @BasePath		/api/v1
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("memuat konfigurasi: %w", err)
	}

	log := logger.New(cfg.App.Env, cfg.App.Debug)

	application, err := app.New(cfg, log)
	if err != nil {
		return fmt.Errorf("bootstrap aplikasi: %w", err)
	}
	defer application.Close()

	// Ctrl+C atau SIGTERM dari orchestrator memicu graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return application.Run(ctx)
}
