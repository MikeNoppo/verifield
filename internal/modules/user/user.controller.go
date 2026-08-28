package user

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/httpx"
	"verifield-be/internal/common/pagination"
	"verifield-be/internal/common/response"
	"verifield-be/internal/modules/user/dto"
)

// Controller memetakan HTTP request ke Service. Padanan *.controller.ts.
// Controller tidak menangani error sendiri: cukup c.Error(err) lalu return,
// dan middleware ErrorHandler yang membentuk response-nya.
type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// FindAll godoc
//
//	@Summary		Daftar user
//	@Description	Mengambil daftar user dengan paginasi, pencarian, dan sorting
//	@Tags			users
//	@Produce		json
//	@Param			page		query		int		false	"Halaman"						default(1)
//	@Param			limit		query		int		false	"Jumlah data per halaman"		default(10)
//	@Param			search		query		string	false	"Cari berdasarkan nama/email"
//	@Param			sort_by		query		string	false	"Kolom sorting"					Enums(created_at, updated_at, name, email, role)
//	@Param			sort_dir	query		string	false	"Arah sorting"					Enums(asc, desc)
//	@Success		200			{object}	response.Envelope{data=[]dto.UserResponse}
//	@Failure		401			{object}	response.Envelope
//	@Failure		403			{object}	response.Envelope
//	@Router			/users [get]
func (ctl *Controller) FindAll(c *gin.Context) {
	var query pagination.Query
	if err := httpx.BindQuery(c, &query); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	users, meta, err := ctl.service.FindAll(c.Request.Context(), query)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.Paginated(c, "Daftar user berhasil diambil", users, meta)
}

// FindOne godoc
//
//	@Summary	Detail user
//	@Tags		users
//	@Produce	json
//	@Param		id	path		string	true	"User ID (UUID)"
//	@Success	200	{object}	response.Envelope{data=dto.UserResponse}
//	@Failure	404	{object}	response.Envelope
//	@Router		/users/{id} [get]
func (ctl *Controller) FindOne(c *gin.Context) {
	user, err := ctl.service.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Detail user berhasil diambil", user)
}

// Create godoc
//
//	@Summary	Buat user baru
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CreateUserDTO	true	"Data user"
//	@Success	201		{object}	response.Envelope{data=dto.UserResponse}
//	@Failure	409		{object}	response.Envelope
//	@Failure	422		{object}	response.Envelope
//	@Router		/users [post]
func (ctl *Controller) Create(c *gin.Context) {
	var payload dto.CreateUserDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	user, err := ctl.service.Create(c.Request.Context(), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.Created(c, "User berhasil dibuat", user)
}

// Update godoc
//
//	@Summary	Perbarui user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"User ID (UUID)"
//	@Param		payload	body		dto.UpdateUserDTO	true	"Field yang ingin diubah"
//	@Success	200		{object}	response.Envelope{data=dto.UserResponse}
//	@Failure	404		{object}	response.Envelope
//	@Router		/users/{id} [patch]
func (ctl *Controller) Update(c *gin.Context) {
	var payload dto.UpdateUserDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	user, err := ctl.service.Update(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "User berhasil diperbarui", user)
}

// Remove godoc
//
//	@Summary	Hapus user
//	@Tags		users
//	@Produce	json
//	@Param		id	path	string	true	"User ID (UUID)"
//	@Success	204	"No Content"
//	@Failure	404	{object}	response.Envelope
//	@Router		/users/{id} [delete]
func (ctl *Controller) Remove(c *gin.Context) {
	if err := ctl.service.Remove(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.NoContent(c)
}
