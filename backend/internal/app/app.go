// Package app adalah padanan AppModule di NestJS: tempat seluruh modul,
// middleware global, dan infrastruktur dirakit menjadi satu aplikasi.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"verifield-be/internal/common/config"
	"verifield-be/internal/common/database"
	"verifield-be/internal/common/middleware"
	"verifield-be/internal/common/validation"
	"verifield-be/internal/modules/joborder"
	"verifield-be/internal/modules/reference"
	"verifield-be/internal/modules/user"
)

// Application memegang seluruh dependency yang hidup selama aplikasi berjalan.
type Application struct {
	cfg    *config.Config
	log    *slog.Logger
	db     *gorm.DB
	engine *gin.Engine

	user      *user.Module
	joborder  *joborder.Module
	reference *reference.Module

	// requireActor menggantikan guard autentikasi selama autentikasi berada di
	// luar cakupan PoC.
	requireActor gin.HandlerFunc
}

// New merakit aplikasi: koneksi database, provider tiap modul, middleware
// global, lalu route. Padanan NestFactory.create(AppModule).
func New(cfg *config.Config, log *slog.Logger) (*Application, error) {
	db, err := database.Connect(cfg.Database)
	if err != nil {
		return nil, err
	}

	// Perubahan schema TIDAK dilakukan di sini. Jalankan `./bin/migrate up`
	// (padanan `prisma migrate deploy`) sebelum menyalakan aplikasi.

	// Pesan validasi memakai nama field JSON, bukan nama field Go.
	validation.Init()

	if cfg.App.IsProduction() || !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// --- Perakitan modul (padanan daftar `imports` di AppModule) ---
	userModule := user.NewModule(db)
	joborderModule := joborder.NewModule(db, userModule.Service)
	referenceModule := reference.NewModule(db)

	engine := gin.New()

	// Urutan middleware penting: RequestID lebih dulu agar tersedia di semua log,
	// ErrorHandler terakhir agar bisa menangkap error dari seluruh handler.
	engine.Use(
		middleware.RequestID(),
		middleware.RequestLogger(log),
		middleware.Recovery(log),
		middleware.CORS(cfg.HTTP.AllowedOrigins),
		middleware.ErrorHandler(log),
	)

	app := &Application{
		cfg:          cfg,
		log:          log,
		db:           db,
		engine:       engine,
		user:         userModule,
		joborder:     joborderModule,
		reference:    referenceModule,
		requireActor: middleware.RequireActor(userModule.Service),
	}

	app.registerRoutes()

	return app, nil
}

// Engine dibuka agar bisa dipakai pada test HTTP (httptest).
func (a *Application) Engine() *gin.Engine { return a.engine }

// Run menjalankan HTTP server dan menutupnya dengan rapi saat ctx dibatalkan.
func (a *Application) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:         a.cfg.HTTP.Addr(),
		Handler:      a.engine,
		ReadTimeout:  a.cfg.HTTP.ReadTimeout,
		WriteTimeout: a.cfg.HTTP.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		a.log.Info("server berjalan",
			"addr", a.cfg.HTTP.Addr(),
			"env", a.cfg.App.Env,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server berhenti: %w", err)
	case <-ctx.Done():
		a.log.Info("sinyal shutdown diterima, menutup server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown gagal: %w", err)
	}

	a.log.Info("server berhenti dengan rapi")
	return nil
}

// Close melepaskan resource yang dipegang aplikasi.
func (a *Application) Close() {
	if err := database.Close(a.db); err != nil {
		a.log.Error("gagal menutup koneksi database", "error", err)
	}
}

// startedAt dipakai endpoint health untuk melaporkan uptime.
var startedAt = time.Now()
