package joborder_test

import (
	"net/http"
	"testing"
	"time"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/modules/joborder"
)

// jakarta adalah zona operasi yang dipakai seluruh test di berkas ini. Aturan
// jam kerja hanya berarti sesuatu bila dinilai pada zona yang benar, dan
// perbedaan tujuh jam terhadap UTC cukup untuk membuat aturan yang salah zona
// terlihat gagal di sini.
var jakarta = time.FixedZone("WIB", 7*60*60)

func policy() joborder.SchedulePolicy {
	return joborder.DefaultSchedulePolicy(jakarta)
}

// at membentuk waktu pada jam tertentu di zona operasi.
func at(day int, hour int) time.Time {
	return time.Date(2026, time.September, day, hour, 0, 0, 0, jakarta)
}

func TestJadwalWajarDiterima(t *testing.T) {
	now := at(1, 9)
	if err := policy().Validate(at(2, 9), at(2, 15), now); err != nil {
		t.Fatalf("jadwal wajar seharusnya diterima, dapat: %v", err)
	}
}

func TestJadwalMasaLaluDitolak(t *testing.T) {
	now := at(10, 9)

	err := policy().Validate(at(1, 9), at(1, 15), now)
	if err == nil {
		t.Fatal("jadwal di masa lalu seharusnya ditolak")
	}
	assertUnprocessable(t, err)
}

// Jadwal yang dipilih "sekarang" lalu terkirim beberapa detik kemudian tidak
// boleh ditolak — kalau ditolak, formulir yang benar akan tampak rusak.
func TestJadwalTerlambatSedikitTetapDiterima(t *testing.T) {
	now := at(1, 9)
	start := now.Add(-2 * time.Minute)

	if err := policy().Validate(start, start.Add(6*time.Hour), now); err != nil {
		t.Fatalf("selisih jam kecil seharusnya ditoleransi, dapat: %v", err)
	}
}

func TestJadwalTerlaluJauhDitolak(t *testing.T) {
	now := at(1, 9)
	start := now.AddDate(1, 0, 0)

	if err := policy().Validate(start, start.Add(6*time.Hour), now); err == nil {
		t.Fatal("jadwal setahun ke depan seharusnya ditolak")
	}
}

func TestJadwalDiLuarJamKerjaDitolak(t *testing.T) {
	now := at(1, 9)

	for _, hour := range []int{3, 7, 18, 23} {
		start := at(2, hour)
		if err := policy().Validate(start, start.Add(2*time.Hour), now); err == nil {
			t.Errorf("pukul %02d.00 di luar jam kerja seharusnya ditolak", hour)
		}
	}
}

// Zona operasi, bukan zona pengirim, yang menentukan. Pukul 02.00 UTC adalah
// pukul 09.00 di lapangan — jadwal yang sah, dan justru bentuk inilah yang
// dikirim peramban karena ia selalu mengirim UTC.
func TestJamKerjaDinilaiPadaZonaOperasi(t *testing.T) {
	now := at(1, 9)
	start := time.Date(2026, time.September, 2, 2, 0, 0, 0, time.UTC)

	if err := policy().Validate(start, start.Add(6*time.Hour), now); err != nil {
		t.Fatalf("02.00 UTC = 09.00 WIB seharusnya diterima, dapat: %v", err)
	}
}

// Kunjungan sore memang berakhir setelah jam kerja. Menolaknya berarti
// melarang jadwal yang sah.
func TestJamSelesaiBolehMelewatiJamKerja(t *testing.T) {
	now := at(1, 9)
	start := at(2, 16)

	if err := policy().Validate(start, start.Add(6*time.Hour), now); err != nil {
		t.Fatalf("jam selesai di luar jam kerja seharusnya diterima, dapat: %v", err)
	}
}

func TestRentangTerlaluPanjangDitolak(t *testing.T) {
	now := at(1, 9)
	start := at(2, 9)

	if err := policy().Validate(start, start.Add(48*time.Hour), now); err == nil {
		t.Fatal("rentang dua hari seharusnya ditolak")
	}
}

func TestJamSelesaiSebelumJamMulaiDitolak(t *testing.T) {
	now := at(1, 9)
	start := at(2, 9)

	if err := policy().Validate(start, start.Add(-time.Hour), now); err == nil {
		t.Fatal("jam selesai sebelum jam mulai seharusnya ditolak")
	}
}

// Penolakan jadwal harus sampai ke klien sebagai kesalahan isian, bukan sebagai
// kesalahan server, dan harus menunjuk field yang salah.
func assertUnprocessable(t *testing.T, err error) {
	t.Helper()

	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("error seharusnya *apperror.AppError, dapat %T", err)
	}
	if appErr.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, mau %d", appErr.Status, http.StatusUnprocessableEntity)
	}
	if len(appErr.Fields) != 1 || appErr.Fields[0].Field != "scheduled_start_at" {
		t.Errorf("field error = %+v, mau menunjuk scheduled_start_at", appErr.Fields)
	}
	if appErr.Message == "" {
		t.Error("pesan penolakan tidak boleh kosong")
	}
}
