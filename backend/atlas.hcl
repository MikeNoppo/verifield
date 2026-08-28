// Konfigurasi Atlas — dipakai HANYA untuk meng-generate berkas migrasi.
// Atlas tidak pernah menyentuh database production; yang menerapkan migrasi
// adalah binary ./cmd/migrate (goose).
//
// Cara kerjanya: Atlas menjalankan ./cmd/atlas-loader untuk tahu schema yang
// diinginkan (dari internal/schema/schema.go), memutar ulang berkas di
// migrations/ pada dev database kosong untuk tahu keadaan sekarang, lalu
// menulis selisihnya sebagai berkas migrasi baru.
//
// Pakai: make migrate-diff name=nama_perubahan

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/atlas-loader",
  ]
}

env "gorm" {
  // Schema yang diinginkan, dibaca dari model GORM.
  src = data.external_schema.gorm.url

  // Dev database: WAJIB database kosong dan terpisah dari database aplikasi.
  // Atlas membuat & menghapus objek di sini untuk menghitung diff.
  // Diisi lewat ATLAS_DEV_URL di .env (lihat .env.example).
  dev = getenv("ATLAS_DEV_URL")

  migration {
    dir = "file://migrations"

    // Berkas ditulis dengan anotasi "-- +goose Up" supaya bisa dibaca goose.
    format = goose
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
