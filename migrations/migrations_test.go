package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// Nama berkas harus mengikuti konvensi goose: <timestamp 14 digit>_<nama>.sql
var fileNamePattern = regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.sql$`)

// TestBerkasMigrasiValid menjaga berkas migrasi tetap sehat tanpa perlu
// database. Ini penting karena Atlas yang meng-generate berkasnya: kalau suatu
// saat Atlas menulis blok Up tanpa blok Down, `migrate down` akan gagal diam-diam
// baru saat dijalankan di production. Test ini menangkapnya lebih awal.
func TestBerkasMigrasiValid(t *testing.T) {
	entries, err := fs.Glob(FS, "*.sql")
	if err != nil {
		t.Fatalf("membaca embed FS: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("tidak ada berkas migrasi yang ter-embed")
	}

	seen := make(map[string]string, len(entries))

	for _, name := range entries {
		t.Run(name, func(t *testing.T) {
			if !fileNamePattern.MatchString(name) {
				t.Errorf("nama berkas tidak sesuai konvensi goose <timestamp>_<nama>.sql")
			}

			version := name[:14]
			if prev, exists := seen[version]; exists {
				t.Errorf("timestamp %s bentrok dengan %s", version, prev)
			}
			seen[version] = name

			raw, err := fs.ReadFile(FS, name)
			if err != nil {
				t.Fatalf("membaca berkas: %v", err)
			}
			content := string(raw)

			up, hasUp := sectionOf(content, "-- +goose Up", "-- +goose Down")
			if !hasUp {
				t.Fatal("tidak ada anotasi '-- +goose Up'")
			}
			if !strings.Contains(content, "-- +goose Down") {
				t.Fatal("tidak ada anotasi '-- +goose Down' — rollback tidak akan bisa dijalankan")
			}
			down, _ := sectionOf(content, "-- +goose Down", "")

			if !hasStatement(up) {
				t.Error("blok Up tidak berisi statement SQL")
			}
			if !hasStatement(down) {
				t.Error("blok Down kosong — `migrate down` akan gagal saat dibutuhkan")
			}
		})
	}
}

// sectionOf mengambil potongan teks setelah penanda start sampai sebelum end.
// end kosong berarti sampai akhir berkas.
func sectionOf(content, start, end string) (string, bool) {
	from := strings.Index(content, start)
	if from < 0 {
		return "", false
	}
	rest := content[from+len(start):]

	if end == "" {
		return rest, true
	}
	if to := strings.Index(rest, end); to >= 0 {
		return rest[:to], true
	}
	return rest, true
}

// hasStatement mengecek ada minimal satu baris SQL sungguhan — bukan sekadar
// komentar atau baris kosong.
func hasStatement(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		return true
	}
	return false
}
