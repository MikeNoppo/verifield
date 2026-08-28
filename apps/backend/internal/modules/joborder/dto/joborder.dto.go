// Package dto berisi kontrak request/response modul job order.
package dto

import (
	"time"

	"verifield-be/internal/common/pagination"
	"verifield-be/internal/schema"
)

// ---------------------------------------------------------------------------
// Request
// ---------------------------------------------------------------------------

// CreateJobOrderDTO adalah body permintaan inspeksi baru dari klien.
// Perusahaan pemesan tidak ikut dikirim — ia diambil dari aktor, supaya klien
// tidak bisa membuat order atas nama perusahaan lain (asumsi A-03).
type CreateJobOrderDTO struct {
	InspectionTypeID  string `json:"inspection_type_id"  binding:"required,uuid4"        example:"6f1e6f0c-6f2a-4c5e-9f3a-0b6b1a4d2c11"`
	ObjectDescription string `json:"object_description"  binding:"required,min=3,max=255" example:"Batu bara 5.000 MT"`
	LocationName      string `json:"location_name"       binding:"required,min=3,max=160" example:"Dermaga 3, Pelabuhan Tanjung Priok"`
	LocationAddress   string `json:"location_address"    binding:"required,min=3"         example:"Jl. Palmerah No. 1, Jakarta Utara"`
	City              string `json:"city"                binding:"required,min=2,max=80"  example:"Jakarta"`

	// Kewajaran jadwal — belum lewat, tidak terlalu jauh ke depan, dan berada
	// pada jam kerja lapangan — diperiksa SchedulePolicy, bukan tag binding:
	// aturannya bergantung pada waktu sekarang dan pada zona operasi, yang
	// keduanya tidak dapat dinyatakan sebagai tag statis.
	ScheduledStartAt time.Time `json:"scheduled_start_at"  binding:"required"`
	ScheduledEndAt   time.Time `json:"scheduled_end_at"    binding:"required,gtfield=ScheduledStartAt"`
}

// AssignInspectorDTO adalah body penugasan inspektor oleh koordinator.
//
// ExpectedVersion wajib: dua koordinator yang menugaskan order yang sama secara
// bersamaan harus menghasilkan satu penolakan yang terlihat, bukan satu
// penugasan yang hilang diam-diam (keputusan B-09).
type AssignInspectorDTO struct {
	InspectorID     string `json:"inspector_id"     binding:"required,uuid4"`
	ExpectedVersion int    `json:"expected_version" binding:"required,min=1" example:"1"`
}

// SubmitStatusEventDTO adalah body pembaruan status dari perangkat inspektor.
type SubmitStatusEventDTO struct {
	ToStatus string `json:"to_status" binding:"required,oneof=on_the_way on_site in_progress completed failed" example:"on_site"`

	// Keputusan B-03: penanda unik dibuat di perangkat sebelum kiriman pertama,
	// sehingga tombol yang ditekan berulang kali saat sinyal lemah hanya
	// menghasilkan satu baris riwayat.
	ClientEventID string `json:"client_event_id" binding:"required,min=8,max=64" example:"6f1e6f0c-6f2a-4c5e-9f3a-0b6b1a4d2c11"`

	// Keputusan B-02: waktu saat tombol ditekan di lapangan, bukan saat
	// laporan berhasil terkirim. Boleh kosong bila perangkat sedang online.
	OccurredAt *time.Time `json:"occurred_at"`

	Reason *string `json:"reason" binding:"omitempty,max=500" example:"Kargo belum tiba di dermaga"`
}

// CancelJobOrderDTO adalah body pembatalan atau pengajuan pembatalan.
type CancelJobOrderDTO struct {
	Reason string `json:"reason" binding:"required,min=3,max=500" example:"Jadwal pengapalan dimajukan"`

	// Kosong berarti pemanggil tidak peduli pada versi. Diisi oleh koordinator
	// yang bertindak dari layar yang mungkin sudah basi.
	ExpectedVersion *int `json:"expected_version" binding:"omitempty,min=1"`
}

// DecideCancellationDTO adalah keputusan koordinator atas permintaan pembatalan.
type DecideCancellationDTO struct {
	Decision string  `json:"decision" binding:"required,oneof=approve reject" example:"approve"`
	Note     *string `json:"note"     binding:"omitempty,max=500"`
}

// SettleCancellationDTO adalah keputusan penyelesaian komersial ketika
// pekerjaan terlanjur selesai mendahului keputusan pembatalan (keputusan B-10).
//
// Note wajib dengan alasan yang sama seperti koreksi: keputusan komersial tanpa
// catatan tidak dapat dipertanggungjawabkan ketika kemudian dipersoalkan, dan
// CS yang menerima telepon klien membaca catatan inilah.
type SettleCancellationDTO struct {
	Outcome string `json:"outcome" binding:"required,oneof=billed_full billed_partial waived" example:"billed_partial"`
	Note    string `json:"note"    binding:"required,min=3,max=500"                          example:"Disepakati menagih separuh biaya kunjungan"`
}

// CorrectStatusDTO adalah koreksi status oleh koordinator, termasuk mundur ke
// tahap sebelumnya.
//
// Reason wajib: koreksi tanpa alasan menghapus nilai audit trail, dan tanpa
// jalur resmi ini koordinator akan mengubah data langsung ke basis data
// (keputusan B-06, alur F-06).
type CorrectStatusDTO struct {
	ToStatus        string `json:"to_status"        binding:"required,oneof=requested assigned on_the_way on_site in_progress completed failed cancelled"`
	Reason          string `json:"reason"           binding:"required,min=3,max=500" example:"Inspektor salah menekan Selesai padahal baru tiba"`
	ExpectedVersion int    `json:"expected_version" binding:"required,min=1"`
}

// ListQuery adalah query string endpoint daftar order.
type ListQuery struct {
	pagination.Query

	Status      string `form:"status"       binding:"omitempty,oneof=requested assigned on_the_way on_site in_progress completed failed cancelled"`
	CompanyID   string `form:"company_id"   binding:"omitempty,uuid4"`
	InspectorID string `form:"inspector_id" binding:"omitempty,uuid4"`

	// Saringan layar koordinator: order mana yang butuh disentuh manusia.
	//   penugasan  — belum ada inspektor
	//   pembatalan — ada permintaan pembatalan menunggu keputusan
	//   basi       — tanpa pembaruan lebih dari 8 jam dan belum final
	//   terlambat  — ada pembaruan terlambat yang ditolak dan belum diselesaikan
	Attention string `form:"attention" binding:"omitempty,oneof=penugasan pembatalan basi terlambat"`
}

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

// JobOrderResponse adalah bentuk job order yang dikirim ke client.
type JobOrderResponse struct {
	ID              string `json:"id"`
	ReferenceNumber string `json:"reference_number" example:"JO-2026-0001"`

	CompanyID     string `json:"company_id"`
	CompanyName   string `json:"company_name"   example:"PT Karya Bahari Ekspor"`
	CreatedByName string `json:"created_by_name"`

	InspectionTypeID   string `json:"inspection_type_id"`
	InspectionTypeName string `json:"inspection_type_name" example:"Inspeksi Kargo Curah"`

	ObjectDescription string `json:"object_description"`
	LocationName      string `json:"location_name"`
	LocationAddress   string `json:"location_address"`
	City              string `json:"city"`

	ScheduledStartAt time.Time `json:"scheduled_start_at"`
	ScheduledEndAt   time.Time `json:"scheduled_end_at"`

	InspectorID   *string `json:"inspector_id"`
	InspectorName *string `json:"inspector_name"`

	CurrentStatus   string    `json:"current_status" enums:"requested,assigned,on_the_way,on_site,in_progress,completed,failed,cancelled" example:"in_progress"`
	StatusChangedAt time.Time `json:"status_changed_at"`

	// Dikirim balik saat mutasi sebagai expected_version (keputusan B-09).
	Version int `json:"version"`

	// Seq event terakhir milik order ini. Klien memakainya untuk mengabaikan
	// pesan real-time yang lebih tua daripada yang sudah diterapkan, sehingga
	// jaminan urutan berlaku di dua sisi — bukan hanya di server.
	Seq int64 `json:"seq"`

	CancellationRequested bool `json:"cancellation_requested"`
	HasOpenAlert          bool `json:"has_open_alert"`

	// Jenis alert yang masih terbuka, karena kalimat yang harus dibaca
	// koordinator berbeda per jenis.
	//
	// NOTE: pekerjaan yang selesai mendahului keputusan pembatalan TIDAK
	// memakai alert. Permintaannya sendiri yang berpindah menunggu penyelesaian
	// dan tetap berada di antrean koordinator (B-10) — satu keadaan cukup satu
	// mekanisme.
	OpenAlertType *string `json:"open_alert_type" enums:"late_update_rejected"`

	// Tahap terjauh yang sempat dicapai sebelum order berakhir. Untuk order yang
	// masih berjalan nilainya sama dengan current_status; untuk yang dibatalkan
	// atau gagal, inilah yang menjawab "sejauh mana sempat berjalan".
	//
	// Dihitung server karena daftar order sengaja tidak membawa riwayat, dan
	// justru di daftar itulah pertanyaan tersebut paling sering muncul.
	ExitStatus *string `json:"exit_status" enums:"requested,assigned,on_the_way,on_site,in_progress,completed,failed,cancelled"`

	// Hanya terisi pada endpoint detail. Koordinator butuh id-nya untuk
	// memutuskan, dan alasannya untuk memutuskan dengan benar.
	PendingCancellation *PendingCancellation `json:"pending_cancellation,omitempty"`

	// Hanya terisi pada endpoint detail; daftar tidak membawa riwayat.
	Events []JobStatusEventResponse `json:"events,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubmitStatusEventResult membungkus hasil pembaruan dari lapangan.
//
// Event yang ditolak tetap menghasilkan 200, bukan error: perangkat sudah
// menyimpan laporan itu secara lokal dan perlu tahu bahwa server sudah
// menerimanya supaya bisa mengeluarkannya dari antrean. Yang gagal adalah
// perubahan status, bukan pengirimannya (keputusan B-07).
type SubmitStatusEventResult struct {
	Accepted  bool                   `json:"accepted"`
	Duplicate bool                   `json:"duplicate"`
	Message   string                 `json:"message"`
	Event     JobStatusEventResponse `json:"event"`
	Order     JobOrderResponse       `json:"order"`
}

// CancelResult membungkus hasil pembatalan, yang bentuknya berbeda tergantung
// tahap: pembatalan langsung, atau permintaan yang menunggu koordinator.
type CancelResult struct {
	Status  string           `json:"status" enums:"cancelled,pending_approval" example:"cancelled"`
	Fee     string           `json:"fee"    enums:"none,travel,visit,coordinator" example:"travel"`
	Message string           `json:"message"`
	Order   JobOrderResponse `json:"order"`
}

// ToJobOrderResponse memetakan entity ke response DTO. Kolom turunan (seq,
// cancellation_requested, has_open_alert) dihitung di repository dan dikirim
// terpisah karena bukan bagian dari tabel job_orders.
func ToJobOrderResponse(o *schema.JobOrder, derived Derived) JobOrderResponse {
	res := JobOrderResponse{
		ID:                    o.ID.String(),
		ReferenceNumber:       o.ReferenceNumber,
		CompanyID:             o.CompanyID.String(),
		InspectionTypeID:      o.InspectionTypeID.String(),
		ObjectDescription:     o.ObjectDescription,
		LocationName:          o.LocationName,
		LocationAddress:       o.LocationAddress,
		City:                  o.City,
		ScheduledStartAt:      o.ScheduledStartAt,
		ScheduledEndAt:        o.ScheduledEndAt,
		CurrentStatus:         string(o.CurrentStatus),
		StatusChangedAt:       o.StatusChangedAt,
		Version:               o.Version,
		Seq:                   derived.Seq,
		CancellationRequested: derived.CancellationRequested,
		HasOpenAlert:          derived.HasOpenAlert,
		OpenAlertType:         derived.OpenAlertType,
		ExitStatus:            derived.ExitStatus,
		PendingCancellation:   derived.PendingCancellation,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
	}

	if o.Company != nil {
		res.CompanyName = o.Company.Name
	}
	if o.CreatedBy != nil {
		res.CreatedByName = o.CreatedBy.Name
	}
	if o.InspectionType != nil {
		res.InspectionTypeName = o.InspectionType.Name
	}
	if o.InspectorID != nil {
		id := o.InspectorID.String()
		res.InspectorID = &id
	}
	if o.Inspector != nil {
		name := o.Inspector.Name
		res.InspectorName = &name
	}
	if len(o.StatusEvents) > 0 {
		res.Events = ToEventResponses(o.StatusEvents)
	}

	return res
}

// Derived adalah kolom turunan hasil subquery di repository.
type Derived struct {
	Seq                   int64
	CancellationRequested bool
	HasOpenAlert          bool
	OpenAlertType         *string
	ExitStatus            *string
	PendingCancellation   *PendingCancellation
}

// PendingCancellation adalah permintaan pembatalan yang masih menunggu
// keputusan koordinator (keputusan B-05).
type PendingCancellation struct {
	ID              string    `json:"id"`
	Reason          string    `json:"reason"`
	RequestedByName string    `json:"requested_by_name"`
	CreatedAt       time.Time `json:"created_at"`

	// Menentukan pertanyaan yang dihadapkan kepada koordinator: pembatalannya
	// sendiri, atau penyelesaian komersial ketika pekerjaan sudah terlanjur
	// selesai (keputusan B-10).
	Status string `json:"status" enums:"pending,pending_settlement" example:"pending"`
}
