package joborder

import (
	"time"

	"verifield-be/internal/schema"
)

// forwardTransitions adalah satu-satunya jalur maju yang sah. Padanannya di
// frontend ada di lib/domain/status.ts, tetapi tabel inilah yang mengikat:
// pembaruan bisa datang dari perangkat yang belum mengetahui status terkini,
// sehingga validasi di antarmuka tidak pernah cukup.
//
// Keputusan B-06: hanya maju. Koreksi mundur bukan transisi, melainkan
// wewenang koordinator lewat jalur terpisah yang wajib beralasan.
var forwardTransitions = map[schema.JobStatus][]schema.JobStatus{
	schema.StatusRequested: {schema.StatusAssigned, schema.StatusCancelled},
	schema.StatusAssigned:  {schema.StatusOnTheWay, schema.StatusCancelled},
	schema.StatusOnTheWay:  {schema.StatusOnSite, schema.StatusCancelled},
	// Failed hanya mungkin setelah inspektor benar-benar tiba — artinya
	// "inspektor tiba, pekerjaan tidak dapat dilaksanakan".
	schema.StatusOnSite:     {schema.StatusInProgress, schema.StatusFailed, schema.StatusCancelled},
	schema.StatusInProgress: {schema.StatusCompleted, schema.StatusFailed, schema.StatusCancelled},
	schema.StatusCompleted:  {},
	schema.StatusFailed:     {},
	schema.StatusCancelled:  {},
}

// CanTransition menjawab apakah perpindahan status diizinkan sebagai transisi
// maju biasa.
func CanTransition(from, to schema.JobStatus) bool {
	for _, allowed := range forwardTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Alasan penolakan yang disimpan pada job_status_events.rejection_reason.
const (
	// RejectionLateAfterFinal dipakai saat pembaruan tiba setelah order
	// berstatus final (keputusan B-07).
	RejectionLateAfterFinal = "late_after_final"
	// RejectionOutOfOrder dipakai saat pembaruan menuntut status yang bukan
	// lanjutan sah dari status sekarang (keputusan B-06).
	RejectionOutOfOrder = "out_of_order"
	// RejectionPendingApproval menandai pembatalan yang diajukan saat pekerjaan
	// sudah berjalan: tercatat di riwayat, tetapi belum mengubah status karena
	// masih menunggu keputusan koordinator (keputusan B-05).
	RejectionPendingApproval = "pending_approval"
	// RejectionCancellationRejected menandai permintaan pembatalan yang ditolak
	// koordinator, sehingga pekerjaan berlanjut.
	RejectionCancellationRejected = "cancellation_rejected"
)

// Batas kewajaran waktu kejadian yang dilaporkan perangkat (keputusan B-02).
// Waktu kejadian berasal dari jam perangkat inspektor, yang bisa saja meleset.
const (
	// maxClockSkewAhead memberi toleransi kecil untuk jam perangkat yang
	// sedikit mendahului server. Lebih dari ini dianggap tidak masuk akal:
	// tidak ada kejadian lapangan yang terjadi di masa depan.
	maxClockSkewAhead = 5 * time.Minute
	// maxBacklogAge membatasi seberapa lama sebuah laporan boleh tertahan di
	// perangkat. Tujuh hari jauh di atas skenario terburuk yang masuk akal
	// (inspektor kehilangan sinyal berhari-hari di lokasi terpencil).
	maxBacklogAge = 7 * 24 * time.Hour
)

// ClampOccurredAt menjaga waktu kejadian tetap berada dalam batas yang masuk
// akal. Nilai di luar batas dijatuhkan ke waktu penerimaan server, dan
// pemanggil menandainya lewat kolom occurred_at_adjusted.
//
// Ini bukan sekadar pembersihan data: timeline yang dilihat klien diurutkan
// berdasarkan waktu kejadian, sehingga satu jam perangkat yang salah setahun
// bisa melempar satu entri ke ujung riwayat dan membuat seluruh urutan tampak
// keliru.
func ClampOccurredAt(occurredAt, receivedAt time.Time) (time.Time, bool) {
	if occurredAt.IsZero() {
		return receivedAt, true
	}
	if occurredAt.After(receivedAt.Add(maxClockSkewAhead)) {
		return receivedAt, true
	}
	if occurredAt.Before(receivedAt.Add(-maxBacklogAge)) {
		return receivedAt, true
	}
	return occurredAt, false
}

// CancelOutcome adalah hasil pembacaan matriks kewenangan pembatalan
// (dokumen konteks bisnis bagian 11).
type CancelOutcome int

const (
	// CancelImmediate berarti order langsung berpindah ke Cancelled.
	CancelImmediate CancelOutcome = iota
	// CancelNeedsApproval berarti permintaan disimpan menunggu keputusan
	// koordinator, dan pekerjaan tetap berjalan (keputusan B-05).
	CancelNeedsApproval
	// CancelForbidden berarti pembatalan tidak diizinkan pada kondisi ini.
	CancelForbidden
)

// CancelFee adalah konsekuensi komersial pembatalan pada tahap tertentu.
type CancelFee string

const (
	FeeNone        CancelFee = "none"
	FeeTravel      CancelFee = "travel"
	FeeVisit       CancelFee = "visit"
	FeeCoordinator CancelFee = "coordinator"
)

// CancelDecision menggabungkan hasil pembacaan matriks beserta kalimat yang
// bisa dimengerti orang non-teknis (F-05). Pesan penolakan ikut di sini supaya
// service tidak menyusun kalimat sendiri di banyak tempat.
type CancelDecision struct {
	Outcome CancelOutcome
	Fee     CancelFee
	Message string
}

// EvaluateCancel membaca matriks kewenangan pembatalan untuk satu kombinasi
// peran dan status.
func EvaluateCancel(role schema.Role, status schema.JobStatus) CancelDecision {
	if status.IsFinal() {
		return CancelDecision{
			Outcome: CancelForbidden,
			Message: "Order ini sudah " + StatusLabel(status) + " dan tidak dapat diubah lagi.",
		}
	}

	switch role {
	case schema.RoleInspector:
		// Keputusan B-04: memisahkan wewenang membatalkan dari pelaksana
		// lapangan adalah kendali internal, sekaligus mencegah terbentuknya
		// insentif menghindari pekerjaan yang sulit atau jauh.
		return CancelDecision{
			Outcome: CancelForbidden,
			Message: "Inspektor tidak berwenang membatalkan order. Bila pekerjaan tidak dapat dilaksanakan, laporkan sebagai kendala beserta alasannya.",
		}
	case schema.RoleCS:
		return CancelDecision{
			Outcome: CancelForbidden,
			Message: "Customer Service hanya memiliki akses baca. Teruskan permintaan ini kepada koordinator.",
		}
	case schema.RoleClient:
		if status == schema.StatusInProgress {
			return CancelDecision{
				Outcome: CancelNeedsApproval,
				Fee:     FeeCoordinator,
				Message: "Pekerjaan sudah dimulai di lokasi. Permintaan pembatalan Anda akan kami teruskan ke koordinator untuk ditinjau.",
			}
		}
	case schema.RoleAdmin:
		// Koordinator boleh membatalkan pada tahap mana pun sebelum final.
	default:
		return CancelDecision{
			Outcome: CancelForbidden,
			Message: "Peran ini tidak berwenang membatalkan order.",
		}
	}

	fee := cancelFeeFor(status)
	return CancelDecision{Outcome: CancelImmediate, Fee: fee, Message: feeMessage(fee)}
}

func cancelFeeFor(status schema.JobStatus) CancelFee {
	switch status {
	case schema.StatusOnTheWay:
		return FeeTravel
	case schema.StatusOnSite:
		return FeeVisit
	case schema.StatusInProgress:
		return FeeCoordinator
	default:
		return FeeNone
	}
}

func feeMessage(fee CancelFee) string {
	switch fee {
	case FeeTravel:
		return "Inspektor sudah dalam perjalanan, sehingga dikenakan biaya perjalanan."
	case FeeVisit:
		return "Inspektor sudah tiba di lokasi, sehingga dikenakan biaya kunjungan."
	case FeeCoordinator:
		return "Biaya ditentukan koordinator."
	default:
		return "Tidak ada biaya pada tahap ini."
	}
}

var statusLabels = map[schema.JobStatus]string{
	schema.StatusRequested:  "diminta",
	schema.StatusAssigned:   "ditugaskan",
	schema.StatusOnTheWay:   "dalam perjalanan",
	schema.StatusOnSite:     "di lokasi",
	schema.StatusInProgress: "sedang dikerjakan",
	schema.StatusCompleted:  "selesai",
	schema.StatusFailed:     "gagal",
	schema.StatusCancelled:  "dibatalkan",
}

// StatusLabel memberi nama status dalam bahasa yang dipakai pesan ke pengguna.
func StatusLabel(status schema.JobStatus) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return string(status)
}
