// Package joborder adalah modul job order: pembuatan, penugasan, pembaruan
// status dari lapangan, pembatalan, dan koreksi.
package joborder

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module memegang seluruh provider modul ini. Service diekspor supaya modul
// realtime bisa menyusun payload dari sumber yang sama dengan endpoint HTTP.
type Module struct {
	Repository Repository
	Service    Service
	Controller *Controller
}

// NewModule merakit repository → service → controller.
//
// users dan schedule di-inject, bukan dikonstruksi di sini: ketergantungan
// lintas modul dan kebijakan yang berasal dari konfigurasi selalu diterima dari
// luar supaya perakitannya terlihat di satu tempat.
func NewModule(db *gorm.DB, users UserFinder, schedule SchedulePolicy) *Module {
	repository := NewRepository(db)
	service := NewService(repository, users, schedule)

	return &Module{
		Repository: repository,
		Service:    service,
		Controller: NewController(service),
	}
}

// RegisterRoutes mendaftarkan route modul ini. Seluruhnya berada di belakang
// guard identitas, karena setiap aksi perlu tahu siapa pelakunya — status
// tercatat atas nama seseorang, dan kepemilikan order dibatasi per perusahaan.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup, guards ...gin.HandlerFunc) {
	orders := rg.Group("/orders", guards...)

	orders.GET("", m.Controller.FindAll)
	orders.POST("", m.Controller.Create)
	orders.GET("/:id", m.Controller.FindOne)

	orders.GET("/:id/events", m.Controller.FindEvents)
	orders.POST("/:id/events", m.Controller.SubmitEvent)

	orders.POST("/:id/assign", m.Controller.Assign)
	orders.POST("/:id/cancel", m.Controller.Cancel)
	orders.POST("/:id/cancellations/:requestId/decide", m.Controller.DecideCancellation)
	orders.POST("/:id/cancellations/:requestId/settle", m.Controller.SettleCancellation)
	orders.POST("/:id/corrections", m.Controller.Correct)
}
