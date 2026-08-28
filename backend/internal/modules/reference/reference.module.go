// Package reference menyajikan data pendukung yang dibutuhkan layar frontend:
// jenis inspeksi untuk form permintaan, daftar inspektor untuk dialog
// penugasan, dan daftar aktor demo sebagai pengganti login.
//
// Dipisahkan dari modul joborder karena isinya murni baca dan tidak memuat satu
// pun aturan bisnis — menaruhnya di sana hanya akan mengaburkan modul yang
// justru paling padat aturan.
package reference

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	Repository Repository
	Controller *Controller
}

func NewModule(db *gorm.DB) *Module {
	repository := NewRepository(db)

	return &Module{
		Repository: repository,
		Controller: NewController(repository),
	}
}

// RegisterRoutes mendaftarkan route modul ini.
//
// Ketiganya sengaja tidak berada di belakang guard identitas: jenis inspeksi
// dan daftar inspektor tidak memuat data komersial klien, dan daftar aktor demo
// justru harus bisa dibaca sebelum ada identitas — ia yang menyediakan
// identitasnya.
func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/inspection-types", m.Controller.InspectionTypes)
	rg.GET("/inspectors", m.Controller.Inspectors)
	rg.GET("/demo/actors", m.Controller.DemoActors)
}
