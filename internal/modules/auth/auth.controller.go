package auth

import (
	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/httpx"
	"verifield-be/internal/common/response"
	"verifield-be/internal/modules/auth/dto"
	"verifield-be/internal/shared/ctxkey"
)

// Controller memetakan endpoint autentikasi ke Service.
type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Register godoc
//
//	@Summary		Registrasi user baru
//	@Description	Mendaftarkan akun baru dengan role `user` dan langsung menerbitkan token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.RegisterDTO	true	"Data registrasi"
//	@Success		201		{object}	response.Envelope{data=dto.AuthResponse}
//	@Failure		409		{object}	response.Envelope
//	@Failure		422		{object}	response.Envelope
//	@Router			/auth/register [post]
func (ctl *Controller) Register(c *gin.Context) {
	var payload dto.RegisterDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	result, err := ctl.service.Register(c.Request.Context(), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.Created(c, "Registrasi berhasil", result)
}

// Login godoc
//
//	@Summary	Login
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.LoginDTO	true	"Kredensial"
//	@Success	200		{object}	response.Envelope{data=dto.AuthResponse}
//	@Failure	401		{object}	response.Envelope
//	@Router		/auth/login [post]
func (ctl *Controller) Login(c *gin.Context) {
	var payload dto.LoginDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	result, err := ctl.service.Login(c.Request.Context(), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Login berhasil", result)
}

// Refresh godoc
//
//	@Summary	Perbarui access token
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.RefreshTokenDTO	true	"Refresh token"
//	@Success	200		{object}	response.Envelope{data=dto.TokenResponse}
//	@Failure	401		{object}	response.Envelope
//	@Router		/auth/refresh [post]
func (ctl *Controller) Refresh(c *gin.Context) {
	var payload dto.RefreshTokenDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	token, err := ctl.service.Refresh(c.Request.Context(), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Token berhasil diperbarui", token)
}

// Me godoc
//
//	@Summary	Profil user yang sedang login
//	@Tags		auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Envelope
//	@Failure	401	{object}	response.Envelope
//	@Router		/auth/me [get]
func (ctl *Controller) Me(c *gin.Context) {
	userID, ok := ctxkey.UserID(c)
	if !ok {
		c.Error(apperror.Unauthorized("Autentikasi diperlukan")) //nolint:errcheck
		return
	}

	profile, err := ctl.service.Profile(c.Request.Context(), userID)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Profil berhasil diambil", profile)
}
