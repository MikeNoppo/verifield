package joborder_test

import (
	"testing"
	"time"

	"verifield-be/internal/modules/joborder"
	"verifield-be/internal/schema"
)

func TestTransisiMajuDiizinkan(t *testing.T) {
	maju := []struct{ from, to schema.JobStatus }{
		{schema.StatusRequested, schema.StatusAssigned},
		{schema.StatusAssigned, schema.StatusOnTheWay},
		{schema.StatusOnTheWay, schema.StatusOnSite},
		{schema.StatusOnSite, schema.StatusInProgress},
		{schema.StatusInProgress, schema.StatusCompleted},
	}

	for _, c := range maju {
		if !joborder.CanTransition(c.from, c.to) {
			t.Errorf("%s -> %s seharusnya diizinkan", c.from, c.to)
		}
	}
}

func TestTransisiMundurDitolak(t *testing.T) {
	// Keputusan B-06: pembaruan yang datang terlambat tidak boleh membuat
	// status yang dilihat klien mundur.
	mundur := []struct{ from, to schema.JobStatus }{
		{schema.StatusInProgress, schema.StatusOnSite},
		{schema.StatusOnSite, schema.StatusOnTheWay},
		{schema.StatusCompleted, schema.StatusInProgress},
		{schema.StatusAssigned, schema.StatusRequested},
	}

	for _, c := range mundur {
		if joborder.CanTransition(c.from, c.to) {
			t.Errorf("%s -> %s seharusnya ditolak", c.from, c.to)
		}
	}
}

func TestTidakAdaJalanKeluarDariStatusFinal(t *testing.T) {
	final := []schema.JobStatus{schema.StatusCompleted, schema.StatusFailed, schema.StatusCancelled}
	semua := []schema.JobStatus{
		schema.StatusRequested, schema.StatusAssigned, schema.StatusOnTheWay,
		schema.StatusOnSite, schema.StatusInProgress, schema.StatusCompleted,
		schema.StatusFailed, schema.StatusCancelled,
	}

	for _, from := range final {
		if !from.IsFinal() {
			t.Fatalf("%s seharusnya berstatus final", from)
		}
		for _, to := range semua {
			if joborder.CanTransition(from, to) {
				t.Errorf("%s final, tetapi transisi ke %s diizinkan", from, to)
			}
		}
	}
}

func TestFailedHanyaSetelahInspektorTiba(t *testing.T) {
	// "Inspektor tiba, pekerjaan tidak dapat dilaksanakan" — sebelum tiba,
	// yang terjadi adalah pembatalan, bukan kegagalan pelaksanaan.
	sebelumTiba := []schema.JobStatus{
		schema.StatusRequested, schema.StatusAssigned, schema.StatusOnTheWay,
	}
	for _, from := range sebelumTiba {
		if joborder.CanTransition(from, schema.StatusFailed) {
			t.Errorf("%s -> failed seharusnya ditolak", from)
		}
	}

	for _, from := range []schema.JobStatus{schema.StatusOnSite, schema.StatusInProgress} {
		if !joborder.CanTransition(from, schema.StatusFailed) {
			t.Errorf("%s -> failed seharusnya diizinkan", from)
		}
	}
}

func TestWaktuKejadianDijepitSaatJamPerangkatMeleset(t *testing.T) {
	received := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		occurred   time.Time
		mauDisetel bool
	}{
		{"wajar, satu jam lalu", received.Add(-time.Hour), false},
		{"tertahan dua hari di perangkat", received.Add(-48 * time.Hour), false},
		{"mendahului sedikit, masih toleran", received.Add(2 * time.Minute), false},
		{"jauh di masa depan", received.Add(72 * time.Hour), true},
		{"jauh di masa lalu", received.Add(-30 * 24 * time.Hour), true},
		// Perangkat online yang tidak mengirim waktu bukan kasus jam meleset.
		// Menandainya sebagai disesuaikan akan memunculkan peringatan palsu
		// pada riwayat yang dibaca klien.
		{"tidak dikirim sama sekali", time.Time{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, adjusted := joborder.ClampOccurredAt(c.occurred, received, time.Time{})

			if adjusted != c.mauDisetel {
				t.Fatalf("adjusted = %v, mau %v", adjusted, c.mauDisetel)
			}
			if adjusted && !got.Equal(received) {
				t.Errorf("saat disetel, waktu harus jatuh ke waktu terima; dapat %v", got)
			}
			want := c.occurred
			if want.IsZero() {
				want = received
			}
			if !adjusted && !got.Equal(want) {
				t.Errorf("saat tidak disetel, waktu asli harus dipertahankan; dapat %v", got)
			}
		})
	}
}

func TestWaktuKejadianTidakBolehMendahuluiOrdernya(t *testing.T) {
	// Laporan tidak mungkin terjadi sebelum ordernya ada. Tanpa batas ini, jam
	// perangkat yang meleset beberapa jam menempatkan "tiba di lokasi" sebelum
	// "order diminta", dan riwayat yang menjadi bukti perselisihan jadi tidak
	// masuk akal.
	createdAt := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	received := createdAt.Add(30 * time.Minute)

	got, adjusted := joborder.ClampOccurredAt(createdAt.Add(-2*time.Hour), received, createdAt)
	if !adjusted {
		t.Error("waktu sebelum order dibuat seharusnya ditandai disesuaikan")
	}
	if !got.Equal(createdAt) {
		t.Errorf("waktu = %v, mau dijepit ke waktu order dibuat %v", got, createdAt)
	}

	after := createdAt.Add(10 * time.Minute)
	got, adjusted = joborder.ClampOccurredAt(after, received, createdAt)
	if adjusted || !got.Equal(after) {
		t.Errorf("waktu sesudah order dibuat harus dipertahankan; dapat %v (adjusted=%v)", got, adjusted)
	}
}

func TestMatriksKewenanganPembatalan(t *testing.T) {
	cases := []struct {
		name   string
		role   schema.Role
		status schema.JobStatus
		want   joborder.CancelOutcome
		fee    joborder.CancelFee
	}{
		{"klien membatalkan sebelum penugasan", schema.RoleClient, schema.StatusRequested, joborder.CancelImmediate, joborder.FeeNone},
		{"klien membatalkan saat inspektor berangkat", schema.RoleClient, schema.StatusOnTheWay, joborder.CancelImmediate, joborder.FeeTravel},
		{"klien membatalkan saat inspektor di lokasi", schema.RoleClient, schema.StatusOnSite, joborder.CancelImmediate, joborder.FeeVisit},
		{"klien membatalkan saat pekerjaan berjalan", schema.RoleClient, schema.StatusInProgress, joborder.CancelNeedsApproval, joborder.FeeCoordinator},
		{"koordinator membatalkan saat pekerjaan berjalan", schema.RoleAdmin, schema.StatusInProgress, joborder.CancelImmediate, joborder.FeeCoordinator},
		{"inspektor tidak pernah boleh membatalkan", schema.RoleInspector, schema.StatusOnSite, joborder.CancelForbidden, ""},
		{"cs hanya membaca", schema.RoleCS, schema.StatusRequested, joborder.CancelForbidden, ""},
		// Wewenang menang atas keadaan: peran yang tidak pernah berwenang
		// mendapat jawaban yang sama pada status apa pun, termasuk yang final.
		{"inspektor tetap tidak berwenang pada order final", schema.RoleInspector, schema.StatusCompleted, joborder.CancelForbidden, ""},
		{"order selesai tidak bisa dibatalkan", schema.RoleAdmin, schema.StatusCompleted, joborder.CancelUnavailable, ""},
		{"order sudah dibatalkan tidak bisa dibatalkan lagi", schema.RoleClient, schema.StatusCancelled, joborder.CancelUnavailable, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := joborder.EvaluateCancel(c.role, c.status)

			if got.Outcome != c.want {
				t.Errorf("outcome = %v, mau %v", got.Outcome, c.want)
			}
			if c.fee != "" && got.Fee != c.fee {
				t.Errorf("fee = %q, mau %q", got.Fee, c.fee)
			}
			if got.Message == "" {
				t.Error("setiap hasil wajib membawa kalimat yang bisa dibaca pengguna")
			}
		})
	}
}

// Penolakan punya dua arah yang berbeda, dan arah itu menentukan tindak lanjut
// yang harus dilakukan inspektor — bukan sekadar kalimat yang lebih enak
// dibaca. Melompat berarti ada tahap yang belum ia laporkan; mundur berarti
// perangkatnya tertinggal dan tidak ada yang perlu ia perbaiki.
func TestArahPenolakanTransisi(t *testing.T) {
	cases := []struct {
		name string
		from schema.JobStatus
		to   schema.JobStatus
		want string
	}{
		{"melompati tahap di depan", schema.StatusOnTheWay, schema.StatusCompleted, joborder.RejectionSkippedStep},
		{"melompat dari penugasan langsung ke kendala", schema.StatusAssigned, schema.StatusFailed, joborder.RejectionSkippedStep},
		{"mundur ke tahap sebelumnya", schema.StatusInProgress, schema.StatusOnSite, joborder.RejectionOutOfOrder},
		{"mengulang tahap yang sedang berlaku", schema.StatusOnTheWay, schema.StatusOnTheWay, joborder.RejectionOutOfOrder},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if joborder.CanTransition(c.from, c.to) {
				t.Fatalf("%s → %s seharusnya bukan transisi yang sah", c.from, c.to)
			}
			if got := joborder.RejectionFor(c.from, c.to); got != c.want {
				t.Errorf("alasan = %q, mau %q", got, c.want)
			}
		})
	}
}
