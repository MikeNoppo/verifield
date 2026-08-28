// Package user adalah modul pengelolaan user.
// Padanan user.module.ts + user.controller.ts + user.service.ts di NestJS.
package user

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module memegang seluruh provider modul ini. Field Service diekspor supaya
// modul lain (mis. auth) bisa memakainya — padanan `exports` di @Module.
type Module struct {
	Repository Repository
	Service    Service
	Controller *Controller
}

// NewModule merakit repository → service → controller.
// Padanan @Module({ providers: [...], controllers: [...] }).
func NewModule(db *gorm.DB) *Module {
	repository := NewRepository(db)
	service := NewService(repository)

	return &Module{
		Repository: repository,
		Service:    service,
		Controller: NewController(service),
	}
}

// RegisterRoutes mendaftarkan route modul ini ke router utama.
//
// WARNING: seluruh route di sini terbuka tanpa guard. Autentikasi berada di luar
// cakupan PoC (dokumen konteks bisnis bagian 13) dan peran dipilih di frontend,
// jadi jangan menyalakan instance ini ke jaringan publik.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")

	users.GET("", m.Controller.FindAll)
	users.POST("", m.Controller.Create)
	users.GET("/:id", m.Controller.FindOne)
	users.PATCH("/:id", m.Controller.Update)
	users.DELETE("/:id", m.Controller.Remove)
}
