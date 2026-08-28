// Package auth menangani registrasi, login, refresh token, dan profil.
// Modul ini bergantung pada user.Service — padanan `imports: [UserModule]`.
package auth

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/modules/user"
	"verifield-be/internal/shared/jwtx"
)

// Module memegang provider modul auth.
type Module struct {
	TokenService *TokenService
	Service      Service
	Controller   *Controller
}

// NewModule merakit token service → auth service → controller.
// user.Service disuntikkan dari luar (oleh app module), bukan dibuat di sini.
func NewModule(manager *jwtx.Manager, users user.Service) *Module {
	tokenService := NewTokenService(manager)
	service := NewService(users, tokenService)

	return &Module{
		TokenService: tokenService,
		Service:      service,
		Controller:   NewController(service),
	}
}

// RegisterRoutes mendaftarkan endpoint auth. Hanya /auth/me yang butuh token.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	group := rg.Group("/auth")

	group.POST("/register", m.Controller.Register)
	group.POST("/login", m.Controller.Login)
	group.POST("/refresh", m.Controller.Refresh)
	group.GET("/me", auth, m.Controller.Me)
}
