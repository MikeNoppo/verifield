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
		nama       string
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
		t.Run(c.nama, func(t *testing.T) {
			got, adjusted := joborder.ClampOccurredAt(c.occurred, received, time.Time{})

			if adjusted != c.mauDisetel {
				t.Fatalf("adjusted = %v, mau %v", adjusted, c.mauDisetel)
			}
			if adjusted && !got.Equal(received) {
				t.Errorf("saat disetel, waktu harus jatuh ke waktu terima; dapat %v", got)
			}
			mau := c.occurred
			if mau.IsZero() {
				mau = received
			}
			if !adjusted && !got.Equal(mau) {
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
	dibuat := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	received := dibuat.Add(30 * time.Minute)

	got, adjusted := joborder.ClampOccurredAt(dibuat.Add(-2*time.Hour), received, dibuat)
	if !adjusted {
		t.Error("waktu sebelum order dibuat seharusnya ditandai disesuaikan")
	}
	if !got.Equal(dibuat) {
		t.Errorf("waktu = %v, mau dijepit ke waktu order dibuat %v", got, dibuat)
	}

	sesudah := dibuat.Add(10 * time.Minute)
	got, adjusted = joborder.ClampOccurredAt(sesudah, received, dibuat)
	if adjusted || !got.Equal(sesudah) {
		t.Errorf("waktu sesudah order dibuat harus dipertahankan; dapat %v (adjusted=%v)", got, adjusted)
	}
}

func TestMatriksKewenanganPembatalan(t *testing.T) {
	cases := []struct {
		nama   string
		role   schema.Role
		status schema.JobStatus
		mau    joborder.CancelOutcome
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
		t.Run(c.nama, func(t *testing.T) {
			got := joborder.EvaluateCancel(c.role, c.status)

			if got.Outcome != c.mau {
				t.Errorf("outcome = %v, mau %v", got.Outcome, c.mau)
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
