package joborder

import (
	"fmt"
	"time"

	"verifield-be/internal/common/apperror"
)

// Batas kewajaran jadwal yang diminta klien.
//
// Formulir sudah membatasi pemilih tanggalnya, tetapi pembatasan itu hanya
// kenyamanan: permintaan bisa datang dari tab yang sudah lama terbuka, dari
// perangkat lain, atau dari integrasi yang tidak pernah melihat formulir sama
// sekali. Aturan yang mengikat harus berada di sini — sama seperti tabel
// transisi status.
const (
	// scheduleGrace menoleransi selisih jam antara peramban klien dan server.
	// Tanpa toleransi ini, jadwal "secepatnya" yang dipilih klien lalu dikirim
	// beberapa detik kemudian akan ditolak sebagai masa lalu.
	scheduleGrace = 5 * time.Minute
	// maxScheduleLead membatasi seberapa jauh permintaan boleh dijadwalkan.
	// Enam bulan jauh di atas praktik nyata; nilai di luar itu hampir selalu
	// salah ketik tahun, dan order yang mengendap sampai kedaluwarsa hanya
	// memenuhi papan koordinator.
	maxScheduleLead = 180 * 24 * time.Hour
	// maxScheduleDuration menjaga satu order tetap berarti satu kunjungan
	// (asumsi A-02). Rentang berhari-hari adalah beberapa penugasan, dan
	// menagihnya sebagai satu penugasan akan salah.
	maxScheduleDuration = 24 * time.Hour
)

// SchedulePolicy adalah jam operasi lapangan.
//
// Zonanya datang dari konfigurasi, bukan dari waktu yang dikirim klien:
// permintaan membawa waktu ber-offset apa pun, sedangkan "jam kerja" hanya
// berarti sesuatu pada zona tempat inspektor benar-benar bekerja (asumsi A-04).
type SchedulePolicy struct {
	Location *time.Location
	OpensAt  int
	ClosesAt int
}

// DefaultSchedulePolicy memakai jam kerja lapangan 08.00–17.00, padanan server
// dari pembatasan yang sama di formulir klien.
func DefaultSchedulePolicy(loc *time.Location) SchedulePolicy {
	if loc == nil {
		loc = time.UTC
	}
	return SchedulePolicy{Location: loc, OpensAt: 8, ClosesAt: 17}
}

// Validate menolak jadwal yang tidak mungkin dieksekusi.
//
// Pesannya ikut dikembalikan sebagai pesan utama, bukan sekadar detail per
// field: klien membaca satu kalimat pada formulirnya, dan kalimat itu harus
// menjelaskan apa yang salah tanpa perlu menerjemahkan kode apa pun (F-05).
func (p SchedulePolicy) Validate(start, end, now time.Time) error {
	if start.Before(now.Add(-scheduleGrace)) {
		return scheduleError("Jadwal yang diminta sudah lewat. Pilih waktu mulai yang belum terlewati.")
	}
	if start.After(now.Add(maxScheduleLead)) {
		return scheduleError(fmt.Sprintf(
			"Jadwal hanya dapat diminta sampai %d hari ke depan. Periksa kembali tanggal yang Anda pilih.",
			int(maxScheduleLead.Hours()/24),
		))
	}

	local := start.In(p.Location)
	if hour := local.Hour(); hour < p.OpensAt || hour > p.ClosesAt {
		// Nama zona diambil dari waktunya sendiri, bukan ditulis tetap:
		// zonanya dapat dikonfigurasi, dan kalimat yang menyebut "WIB" pada
		// deployment berzona lain justru menyesatkan.
		return scheduleError(fmt.Sprintf(
			"Jadwal hanya dapat dimulai pada jam kerja, pukul %02d.00–%02d.00 %s. Di luar jam itu tidak ada inspektor maupun pihak ketiga di lokasi.",
			p.OpensAt, p.ClosesAt, local.Format("MST"),
		))
	}

	// Jam selesai sengaja TIDAK diperiksa terhadap jam kerja: satu kunjungan
	// yang dimulai sore hari memang berakhir setelah jam kerja, dan menolaknya
	// akan melarang jadwal yang sah.
	if !end.After(start) {
		return scheduleError("Jam selesai harus berada setelah jam mulai.")
	}
	if end.Sub(start) > maxScheduleDuration {
		return scheduleError("Satu permintaan mencakup satu kunjungan, paling lama 24 jam. Pekerjaan yang lebih panjang dipecah menjadi beberapa permintaan.")
	}

	return nil
}

func scheduleError(message string) error {
	return apperror.UnprocessableEntity(message).WithFields([]apperror.FieldError{
		{Field: "scheduled_start_at", Message: message},
	})
}
