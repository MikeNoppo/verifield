package dto

import (
	"time"

	"verifield-be/internal/schema"
)

// JobStatusEventResponse adalah satu entri riwayat status.
//
// Seluruh kolom audit ikut dikirim, termasuk waktu penerimaan dan penanda unik
// perangkat. Penyaringan mana yang boleh dilihat klien dilakukan di antarmuka,
// bukan dengan menghilangkan datanya dari sini — koordinator dan klien membaca
// endpoint yang sama, dan riwayat inilah yang menjadi bukti bila kemudian
// terjadi perselisihan.
type JobStatusEventResponse struct {
	ID  string `json:"id"  example:"6f1e6f0c-6f2a-4c5e-9f3a-0b6b1a4d2c11"`
	Seq int64  `json:"seq" example:"42"`

	JobOrderID string  `json:"job_order_id"`
	FromStatus *string `json:"from_status" enums:"requested,assigned,on_the_way,on_site,in_progress,completed,failed,cancelled" example:"on_site"`
	ToStatus   string  `json:"to_status"   enums:"requested,assigned,on_the_way,on_site,in_progress,completed,failed,cancelled" example:"in_progress"`

	ActorID   *string `json:"actor_id"`
	ActorName string  `json:"actor_name" example:"Rina Amelia"`
	ActorRole string  `json:"actor_role" enums:"admin,client,inspector,cs" example:"inspector"`

	// Keputusan B-02: waktu kejadian di lapangan dan waktu terima server dicatat
	// terpisah. Timeline diurutkan dengan OccurredAt; status terkini ditentukan
	// dengan Seq.
	OccurredAt         time.Time `json:"occurred_at"`
	ReceivedAt         time.Time `json:"received_at"`
	OccurredAtAdjusted bool      `json:"occurred_at_adjusted"`

	ClientEventID *string `json:"client_event_id"`

	// Keputusan B-07: event yang ditolak tetap tersimpan. Hanya event
	// Accepted=true yang pernah mengubah status.
	Accepted        bool    `json:"accepted"`
	RejectionReason *string `json:"rejection_reason" enums:"late_after_final,out_of_order,skipped_step,pending_approval,cancellation_rejected,settlement_pending,settlement_decided" example:"late_after_final"`

	IsCorrection bool    `json:"is_correction"`
	Reason       *string `json:"reason"`

	CreatedAt time.Time `json:"created_at"`
}

// ToEventResponse memetakan entity ke response DTO. Nama aktor diambil dari
// relasi Actor bila sudah di-preload; event buatan sistem tidak punya aktor.
func ToEventResponse(e *schema.JobStatusEvent) JobStatusEventResponse {
	res := JobStatusEventResponse{
		ID:                 e.ID.String(),
		Seq:                e.Seq,
		JobOrderID:         e.JobOrderID.String(),
		ToStatus:           string(e.ToStatus),
		ActorRole:          string(e.ActorRole),
		OccurredAt:         e.OccurredAt,
		ReceivedAt:         e.ReceivedAt,
		OccurredAtAdjusted: e.OccurredAtAdjusted,
		ClientEventID:      e.ClientEventID,
		Accepted:           e.Accepted,
		RejectionReason:    e.RejectionReason,
		IsCorrection:       e.IsCorrection,
		Reason:             e.Reason,
		CreatedAt:          e.CreatedAt,
	}

	if e.FromStatus != nil {
		from := string(*e.FromStatus)
		res.FromStatus = &from
	}
	if e.ActorID != nil {
		id := e.ActorID.String()
		res.ActorID = &id
	}
	if e.Actor != nil {
		res.ActorName = e.Actor.Name
	}

	return res
}

// ToEventResponses memetakan slice entity ke slice response DTO.
func ToEventResponses(events []schema.JobStatusEvent) []JobStatusEventResponse {
	out := make([]JobStatusEventResponse, 0, len(events))
	for i := range events {
		out = append(out, ToEventResponse(&events[i]))
	}
	return out
}
