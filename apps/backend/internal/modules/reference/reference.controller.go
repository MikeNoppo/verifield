package reference

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/response"
)

type Controller struct {
	repo Repository
}

func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
}

// InspectionTypes godoc
//
//	@Summary	Daftar jenis inspeksi yang masih ditawarkan
//	@Tags		reference
//	@Produce	json
//	@Success	200	{object}	response.Envelope{data=[]reference.InspectionTypeResponse}
//	@Router		/inspection-types [get]
func (ctl *Controller) InspectionTypes(c *gin.Context) {
	types, err := ctl.repo.InspectionTypes(c.Request.Context())
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}
	response.OK(c, "Daftar jenis inspeksi berhasil diambil", types)
}

// Inspectors godoc
//
//	@Summary	Daftar inspektor aktif beserta jumlah penugasan berjalan
//	@Tags		reference
//	@Produce	json
//	@Success	200	{object}	response.Envelope{data=[]reference.InspectorResponse}
//	@Router		/inspectors [get]
func (ctl *Controller) Inspectors(c *gin.Context) {
	inspectors, err := ctl.repo.Inspectors(c.Request.Context())
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}
	response.OK(c, "Daftar inspektor berhasil diambil", inspectors)
}

// DemoActors godoc
//
//	@Summary		Identitas siap pakai untuk pemilih peran
//	@Description	Pengganti login selama autentikasi berada di luar cakupan PoC. Id yang dikembalikan dipakai sebagai header X-Actor-Id.
//	@Tags			reference
//	@Produce		json
//	@Success		200	{object}	response.Envelope{data=[]reference.DemoActorResponse}
//	@Router			/demo/actors [get]
func (ctl *Controller) DemoActors(c *gin.Context) {
	actors, err := ctl.repo.DemoActors(c.Request.Context())
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}
	response.OK(c, "Daftar aktor demo berhasil diambil", actors)
}
