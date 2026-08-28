package joborder

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/httpx"
	"verifield-be/internal/common/response"
	"verifield-be/internal/modules/joborder/dto"
	"verifield-be/internal/shared/ctxkey"
)

// Controller memetakan HTTP request ke Service.
// Controller tidak menangani error sendiri: cukup c.Error(err) lalu return,
// dan middleware ErrorHandler yang membentuk response-nya.
type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// actorOf menerjemahkan pengguna hasil middleware menjadi Actor domain.
func actorOf(c *gin.Context) (Actor, error) {
	user, ok := ctxkey.Actor(c)
	if !ok {
		return Actor{}, apperror.Unauthorized("Identitas pemanggil tidak dikenali")
	}
	return Actor{
		ID:        user.ID,
		Name:      user.Name,
		Role:      user.Role,
		CompanyID: user.CompanyID,
	}, nil
}

// FindAll godoc
//
//	@Summary		Daftar job order
//	@Description	Klien otomatis hanya melihat order milik perusahaannya sendiri, apa pun isi company_id.
//	@Tags			job-orders
//	@Produce		json
//	@Param			X-Actor-Id		header		string	true	"Id pengguna yang bertindak"
//	@Param			page			query		int		false	"Halaman"					default(1)
//	@Param			limit			query		int		false	"Jumlah data per halaman"	default(10)
//	@Param			search			query		string	false	"Cari ref/objek/lokasi/kota"
//	@Param			status			query		string	false	"Saring status"	Enums(requested, assigned, on_the_way, on_site, in_progress, completed, failed, cancelled)
//	@Param			company_id		query		string	false	"Saring perusahaan"
//	@Param			inspector_id	query		string	false	"Saring inspektor"
//	@Param			attention		query		string	false	"Saringan layar koordinator"	Enums(penugasan, pembatalan, basi, terlambat)
//	@Success		200				{object}	response.Envelope{data=[]dto.JobOrderResponse}
//	@Router			/orders [get]
func (ctl *Controller) FindAll(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var query dto.ListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	orders, meta, err := ctl.service.List(c.Request.Context(), actor, query)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.Paginated(c, "Daftar job order berhasil diambil", orders, meta)
}

// FindOne godoc
//
//	@Summary	Detail job order beserta riwayat status
//	@Tags		job-orders
//	@Produce	json
//	@Param		X-Actor-Id	header		string	true	"Id pengguna yang bertindak"
//	@Param		id			path		string	true	"Job order ID (UUID)"
//	@Success	200			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Failure	404			{object}	response.Envelope
//	@Router		/orders/{id} [get]
func (ctl *Controller) FindOne(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.Detail(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Detail job order berhasil diambil", order)
}

// FindEvents godoc
//
//	@Summary		Riwayat status satu job order
//	@Description	after_seq memutar ulang hanya perubahan yang lebih baru dari kursor yang dipegang klien.
//	@Tags			job-orders
//	@Produce		json
//	@Param			X-Actor-Id	header		string	true	"Id pengguna yang bertindak"
//	@Param			id			path		string	true	"Job order ID (UUID)"
//	@Param			after_seq	query		int		false	"Kursor seq terakhir yang sudah dimiliki klien"
//	@Success		200			{object}	response.Envelope{data=[]dto.JobStatusEventResponse}
//	@Router			/orders/{id}/events [get]
func (ctl *Controller) FindEvents(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	afterSeq, err := parseSeqQuery(c, "after_seq")
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	events, err := ctl.service.Events(c.Request.Context(), actor, c.Param("id"), afterSeq)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Riwayat status berhasil diambil", events)
}

// Create godoc
//
//	@Summary	Buat permintaan inspeksi baru
//	@Tags		job-orders
//	@Accept		json
//	@Produce	json
//	@Param		X-Actor-Id	header		string					true	"Id pengguna yang bertindak"
//	@Param		payload		body		dto.CreateJobOrderDTO	true	"Data permintaan"
//	@Success	201			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Failure	422			{object}	response.Envelope
//	@Router		/orders [post]
func (ctl *Controller) Create(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.CreateJobOrderDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.Create(c.Request.Context(), actor, payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.Created(c, "Permintaan inspeksi berhasil dibuat", order)
}

// Assign godoc
//
//	@Summary		Tugaskan inspektor
//	@Description	expected_version wajib; bila order sudah berubah, permintaan ditolak dengan 409 (keputusan B-09).
//	@Tags			job-orders
//	@Accept			json
//	@Produce		json
//	@Param			X-Actor-Id	header		string					true	"Id pengguna yang bertindak"
//	@Param			id			path		string					true	"Job order ID (UUID)"
//	@Param			payload		body		dto.AssignInspectorDTO	true	"Inspektor dan versi yang diharapkan"
//	@Success		200			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Failure		409			{object}	response.Envelope
//	@Router			/orders/{id}/assign [post]
func (ctl *Controller) Assign(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.AssignInspectorDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.Assign(c.Request.Context(), actor, c.Param("id"), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Inspektor berhasil ditugaskan", order)
}

// SubmitEvent godoc
//
//	@Summary		Perbarui status dari lapangan
//	@Description	Idempoten lewat client_event_id. Pembaruan yang ditolak tetap menghasilkan 200 karena pengirimannya berhasil — yang gagal adalah perubahan statusnya (keputusan B-07).
//	@Tags			job-orders
//	@Accept			json
//	@Produce		json
//	@Param			X-Actor-Id	header		string						true	"Id pengguna yang bertindak"
//	@Param			id			path		string						true	"Job order ID (UUID)"
//	@Param			payload		body		dto.SubmitStatusEventDTO	true	"Status baru beserta penanda unik perangkat"
//	@Success		200			{object}	response.Envelope{data=dto.SubmitStatusEventResult}
//	@Router			/orders/{id}/events [post]
func (ctl *Controller) SubmitEvent(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.SubmitStatusEventDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	result, err := ctl.service.SubmitEvent(c.Request.Context(), actor, c.Param("id"), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, result.Message, result)
}

// Cancel godoc
//
//	@Summary		Batalkan atau ajukan pembatalan
//	@Description	Hasilnya bergantung pada tahap dan peran. Setelah pekerjaan dimulai, pembatalan oleh klien menjadi permintaan yang menunggu koordinator (keputusan B-05).
//	@Tags			job-orders
//	@Accept			json
//	@Produce		json
//	@Param			X-Actor-Id	header		string					true	"Id pengguna yang bertindak"
//	@Param			id			path		string					true	"Job order ID (UUID)"
//	@Param			payload		body		dto.CancelJobOrderDTO	true	"Alasan pembatalan"
//	@Success		200			{object}	response.Envelope{data=dto.CancelResult}
//	@Failure		409			{object}	response.Envelope
//	@Router			/orders/{id}/cancel [post]
func (ctl *Controller) Cancel(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.CancelJobOrderDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	result, err := ctl.service.Cancel(c.Request.Context(), actor, c.Param("id"), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, result.Message, result)
}

// DecideCancellation godoc
//
//	@Summary	Putuskan permintaan pembatalan
//	@Tags		job-orders
//	@Accept		json
//	@Produce	json
//	@Param		X-Actor-Id	header		string						true	"Id pengguna yang bertindak"
//	@Param		id			path		string						true	"Job order ID (UUID)"
//	@Param		requestId	path		string						true	"Cancellation request ID (UUID)"
//	@Param		payload		body		dto.DecideCancellationDTO	true	"Keputusan koordinator"
//	@Success	200			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Router		/orders/{id}/cancellations/{requestId}/decide [post]
func (ctl *Controller) DecideCancellation(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.DecideCancellationDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.DecideCancellation(
		c.Request.Context(), actor, c.Param("id"), c.Param("requestId"), payload,
	)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Keputusan pembatalan tercatat", order)
}

// SettleCancellation godoc
//
//	@Summary		Putuskan penyelesaian komersial
//	@Description	Dipakai ketika pekerjaan terlanjur selesai mendahului keputusan pembatalan. Status order tidak berubah; yang dicatat adalah siapa menanggung biayanya (keputusan B-10).
//	@Tags			job-orders
//	@Accept			json
//	@Produce		json
//	@Param			X-Actor-Id	header		string						true	"Id pengguna yang bertindak"
//	@Param			id			path		string						true	"Job order ID (UUID)"
//	@Param			requestId	path		string						true	"Cancellation request ID (UUID)"
//	@Param			payload		body		dto.SettleCancellationDTO	true	"Hasil penyelesaian beserta catatannya"
//	@Success		200			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Router			/orders/{id}/cancellations/{requestId}/settle [post]
func (ctl *Controller) SettleCancellation(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.SettleCancellationDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.SettleCancellation(
		c.Request.Context(), actor, c.Param("id"), c.Param("requestId"), payload,
	)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Penyelesaian tercatat", order)
}

// Correct godoc
//
//	@Summary		Koreksi status oleh koordinator
//	@Description	Boleh mundur ke tahap sebelumnya, wajib beralasan, dan tercatat sebagai entri baru — riwayat tidak pernah ditimpa (keputusan B-06).
//	@Tags			job-orders
//	@Accept			json
//	@Produce		json
//	@Param			X-Actor-Id	header		string					true	"Id pengguna yang bertindak"
//	@Param			id			path		string					true	"Job order ID (UUID)"
//	@Param			payload		body		dto.CorrectStatusDTO	true	"Status hasil koreksi beserta alasannya"
//	@Success		200			{object}	response.Envelope{data=dto.JobOrderResponse}
//	@Router			/orders/{id}/corrections [post]
func (ctl *Controller) Correct(c *gin.Context) {
	actor, err := actorOf(c)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	var payload dto.CorrectStatusDTO
	if err := httpx.BindJSON(c, &payload); err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	order, err := ctl.service.Correct(c.Request.Context(), actor, c.Param("id"), payload)
	if err != nil {
		c.Error(err) //nolint:errcheck
		return
	}

	response.OK(c, "Status berhasil dikoreksi", order)
}

func parseSeqQuery(c *gin.Context, name string) (int64, error) {
	raw := c.Query(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, apperror.BadRequest("Parameter " + name + " harus berupa bilangan bulat tidak negatif")
	}
	return value, nil
}
