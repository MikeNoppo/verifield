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
// Parameter guard adalah middleware yang dipasang per route,
// padanan @UseGuards(JwtAuthGuard, RolesGuard).
func (m *Module) RegisterRoutes(rg *gin.RouterGroup, auth, adminOnly gin.HandlerFunc) {
	users := rg.Group("/users", auth)

	users.GET("", adminOnly, m.Controller.FindAll)
	users.POST("", adminOnly, m.Controller.Create)
	users.GET("/:id", m.Controller.FindOne)
	users.PATCH("/:id", adminOnly, m.Controller.Update)
	users.DELETE("/:id", adminOnly, m.Controller.Remove)
}
