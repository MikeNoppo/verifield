// Package realtime mengirim perubahan job order ke klien lewat Server-Sent
// Events, dan menyebarkannya antar instance lewat LISTEN/NOTIFY Postgres.
//
// SSE dipilih, bukan WebSocket, karena klien hanya membaca — dan karena SSE
// membawa dua hal yang justru paling dibutuhkan di sini sebagai bagian dari
// protokolnya: penyambungan ulang otomatis, dan header Last-Event-ID yang
// berpasangan persis dengan kolom seq pada job_status_events.
package realtime

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Module struct {
	Hub        *Hub
	Controller *Controller

	listener *Listener
}

// NewModule merakit hub, listener, dan controller.
//
// dsn dipisahkan dari koneksi GORM dengan sengaja: listener memerlukan koneksi
// tersendiri di luar connection pool.
func NewModule(dsn string, orders SnapshotProvider, log *slog.Logger) *Module {
	hub := NewHub()

	return &Module{
		Hub:        hub,
		Controller: NewController(hub, orders),
		listener:   NewListener(dsn, hub, orders, log),
	}
}

// Start menjalankan listener di latar belakang sampai ctx dibatalkan.
func (m *Module) Start(ctx context.Context) {
	go m.listener.Run(ctx)
}

// RegisterRoutes mendaftarkan endpoint stream di belakang guard identitas —
// cakupan siaran ditentukan peran pemanggil, jadi identitasnya wajib diketahui.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup, guards ...gin.HandlerFunc) {
	rg.GET("/stream", append(guards, m.Controller.Stream)...)
}
