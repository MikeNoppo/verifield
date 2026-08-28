package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"verifield-be/internal/common/response"
)

// apiPrefix adalah global prefix, padanan app.setGlobalPrefix('api/v1').
const apiPrefix = "/api/v1"

// registerRoutes memasang seluruh route aplikasi.
// Menambah modul baru cukup satu baris RegisterRoutes di bawah.
func (a *Application) registerRoutes() {
	a.engine.GET("/health", healthHandler)
	a.engine.GET("/ready", a.readyHandler)

	api := a.engine.Group(apiPrefix)
	api.GET("/health", healthHandler)
	api.GET("/ready", a.readyHandler)

	a.user.RegisterRoutes(api)
	a.reference.RegisterRoutes(api)
	a.joborder.RegisterRoutes(api, a.requireActor)
	a.realtime.RegisterRoutes(api, a.requireActor)

	// Dokumentasi API tidak diekspos di production.
	if !a.cfg.App.IsProduction() {
		a.engine.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	}

	a.engine.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan", nil)
	})

	a.engine.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method tidak diizinkan", nil)
	})
}

// healthHandler godoc
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/health [get]
func healthHandler(c *gin.Context) {
	response.OK(c, "Service berjalan normal", gin.H{
		"status":    "ok",
		"uptime":    time.Since(startedAt).String(),
		"timestamp": time.Now().UTC(),
	})
}

// readyHandler godoc
//
//	@Summary		Readiness probe
//	@Description	Berbeda dari health check: probe ini menyentuh database, dan menjawab 503 selama database belum terjangkau.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	response.Envelope
//	@Failure		503	{object}	response.Envelope
//	@Router			/ready [get]
//
// WARNING: jangan pakai endpoint ini sebagai livenessProbe. Liveness yang ikut
// memeriksa database akan membunuh dan menyalakan ulang seluruh pod ketika yang
// bermasalah justru databasenya — memperparah keadaan, bukan memulihkannya.
// Liveness memakai /health, yang sengaja tidak menyentuh dependensi apa pun.
func (a *Application) readyHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	sqlDB, err := a.db.DB()
	if err == nil {
		err = sqlDB.PingContext(ctx)
	}
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "NOT_READY",
			"Database belum terjangkau", nil)
		return
	}

	response.OK(c, "Siap menerima lalu lintas", gin.H{"status": "ready"})
}
