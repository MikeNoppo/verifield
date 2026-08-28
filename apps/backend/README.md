# Verifield BE

REST API layanan inspeksi & sampling lapangan dengan **Go + [Gin](https://github.com/gin-gonic/gin) + GORM (PostgreSQL)**, disusun memakai struktur modular ala **NestJS** — tiap fitur berdiri sendiri di satu folder berisi controller, service, repository, dan DTO. Schema database terpusat di satu file ala **Prisma**.

## Peta konsep NestJS → project ini

| NestJS | Di project ini |
|---|---|
| `main.ts` | [cmd/api/main.go](cmd/api/main.go) |
| `app.module.ts` | [internal/app/app.go](internal/app/app.go) |
| `xxx.module.ts` | `internal/modules/xxx/xxx.module.go` |
| `xxx.controller.ts` | `internal/modules/xxx/xxx.controller.go` |
| `xxx.service.ts` | `internal/modules/xxx/xxx.service.go` |
| Repository TypeORM | `internal/modules/xxx/xxx.repository.go` |
| `dto/*.dto.ts` + class-validator | `internal/modules/xxx/dto/*.dto.go` + tag `binding:"..."` |
| `schema.prisma` (Prisma) | [internal/schema/schema.go](internal/schema/schema.go) — semua tabel dalam satu file |
| `TransformInterceptor` | [internal/common/response](internal/common/response/response.go) |
| `HttpException` + `ExceptionFilter` | [apperror](internal/common/apperror/apperror.go) + [error_handler.go](internal/common/middleware/error_handler.go) |
| `ConfigModule` | [internal/common/config](internal/common/config/config.go) |
| `ValidationPipe` | [httpx.Bind*](internal/common/httpx/bind.go) + [validation](internal/common/validation/validation.go) |

## Struktur folder

```
cmd/
  api/                      entrypoint aplikasi
  migrate/                  penerap migrasi (goose, migrasi ter-embed)
  seeder/                   pembuat admin pertama
  atlas-loader/             pencetak DDL dari schema, dipanggil Atlas
internal/
  app/                      AppModule: perakitan modul + router global
  common/                   infrastruktur lintas modul
    config/ database/ logger/
    apperror/ response/ validation/ httpx/ pagination/
    middleware/             recovery, request logger, CORS, error handler
  schema/                   SCHEMA DATABASE — semua tabel & relasi dalam satu file
  modules/
    user/                   modul user — CRUD lengkap, pakai sebagai template modul baru
  shared/                   hash, ctxkey
migrations/                 berkas migrasi SQL (ter-embed ke binary)
docs/                       spesifikasi Swagger (hasil generate)
atlas.hcl                   konfigurasi generate migrasi
```

## Menjalankan

```bash
cp .env.example .env      # sesuaikan DB_*
go mod tidy
make migrate-up           # buat tabel  (~ prisma migrate deploy)
make seed                 # buat admin pertama  (~ prisma db seed)
make run                  # atau: go run ./cmd/api
```

Butuh PostgreSQL aktif. Schema **tidak** lagi dibuat otomatis saat startup — lihat [Migrasi](#migrasi).

Perintah lain: `make help`.

## Endpoint

Global prefix `/api/v1`.

| Method | Path |
|---|---|
| GET | `/health` |
| GET · POST | `/users` |
| GET · PATCH · DELETE | `/users/:id` |

> **Seluruh endpoint terbuka tanpa autentikasi.** Autentikasi dan manajemen pengguna berada di luar cakupan PoC (dokumen konteks bisnis bagian 13); peran dipilih di halaman `/` frontend, bukan lewat login. Jangan menyalakan instance ini ke jaringan publik.

Dokumentasi interaktif: `make swagger` lalu buka <http://localhost:8080/swagger/index.html> (nonaktif saat `APP_ENV=production`).

### Contoh

```bash
curl localhost:8080/api/v1/users

curl -X POST localhost:8080/api/v1/users -H 'Content-Type: application/json' \
  -d '{"name":"Siti Rahma","email":"siti@verifield.id","password":"rahasia123","role":"client"}'
```

Role yang tersedia: `admin` (koordinator operasional), `client`, `inspector`, `cs`. Admin pertama dibuat lewat `make seed` (baca `SEED_ADMIN_*` dari `.env`), yang sekaligus mengisi data contoh.

## Bentuk response

Semua endpoint memakai envelope yang sama.

```jsonc
// sukses
{ "success": true, "message": "Login berhasil", "data": { } }

// list berpaginasi
{ "success": true, "message": "...", "data": [], "meta": { "page": 1, "limit": 10, "total": 42, "total_page": 5 } }

// gagal validasi (422)
{ "success": false, "message": "Validasi gagal", "code": "VALIDATION_ERROR",
  "errors": [{ "field": "email", "message": "email harus berupa alamat email yang valid" }] }
```

Endpoint list menerima `?page=&limit=&search=&sort_by=&sort_dir=`. Kolom `sort_by` difilter whitelist (`SortableColumns` di tiap repository) agar aman dari SQL injection.

## Schema database

Seluruh schema ada di **satu file**: [internal/schema/schema.go](internal/schema/schema.go) — padanan `schema.prisma`. Semua tabel, kolom, index, dan relasi terkumpul di sana, tidak tersebar per modul.

```go
type User struct {
    ID    uuid.UUID `gorm:"type:uuid;primaryKey"`
    Email string    `gorm:"type:varchar(160);not null;uniqueIndex"`
    Role  Role      `gorm:"type:varchar(20);not null;default:client"`
}
```

Struct di file ini adalah **sumber kebenaran**; perubahannya diterjemahkan jadi file SQL bernomor lewat [Migrasi](#migrasi) di bawah.

Relasi ditulis di file yang sama, jadi kedua sisinya terlihat berdampingan:

```go
// belongs-to
OwnerID uuid.UUID `gorm:"type:uuid;not null;index"`
Owner   *User     `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`

// has-many
Products []Product `gorm:"foreignKey:OwnerID"`

// many-to-many
Tags []Tag `gorm:"many2many:product_tags"`
```

Relasi **tidak** ikut ter-load otomatis. Minta eksplisit di repository — padanan `include` di Prisma:

```go
db.Preload("Owner").Find(&products)
```

## Migrasi

Pembagian tugasnya: **Atlas meng-generate** SQL dari `schema.go`, **goose menerapkannya**.

| Prisma | Di sini |
|---|---|
| `prisma migrate dev --name add_products` | `make migrate-diff name=add_products` |
| `prisma migrate deploy` | `make migrate-up` (di server: `./bin/migrate up`) |
| `prisma migrate status` | `make migrate-status` |
| rollback | `make migrate-down` |
| `prisma db seed` | `make seed` |

Atlas **tidak pernah menyentuh database production** — ia hanya membaca `schema.go`, memutar ulang berkas di [migrations/](migrations/) pada sebuah dev database kosong, lalu menulis selisihnya sebagai berkas baru. Yang menyentuh production hanya `bin/migrate`, dan berkas migrasinya sudah **ter-embed di dalam binary** ([migrations/embed.go](migrations/embed.go)) — jadi server tidak perlu Atlas, tidak perlu folder `migrations/`, dan tidak perlu Go toolchain.

### Persiapan sekali saja

```bash
go install ariga.io/atlas/cmd/atlas@latest
createdb verifield_dev      # dev database KOSONG, terpisah dari database aplikasi
                            # lalu isi ATLAS_DEV_URL di .env
atlas migrate hash --env gorm   # buat atlas.sum untuk migrasi baseline
```

### Alur harian

```bash
# 1. ubah internal/schema/schema.go
# 2. generate migrasinya
make migrate-diff name=add_products
# 3. baca berkas SQL yang muncul di migrations/, pastikan blok Down-nya benar
# 4. terapkan & commit
make migrate-up
```

Kalau Anda mengedit berkas migrasi dengan tangan, jalankan `make migrate-hash` supaya checksum `atlas.sum` kembali cocok.

### Urutan deploy

```bash
make build          # bin/verifield-be, bin/migrate, bin/seeder
./bin/migrate up    # ~ prisma migrate deploy
./bin/verifield-be  # start
```

Migrasi sengaja **tidak** dijalankan otomatis saat aplikasi start, supaya beberapa instance yang menyala bersamaan tidak saling berebut mengubah schema.

> Atlas menghitung rollback secara dinamis, sedangkan goose membacanya dari blok `-- +goose Down` di berkas. Kalau hasil generate ternyata tidak menyertakan blok itu, tulis sendiri lalu jalankan `make migrate-hash`. [migrations/migrations_test.go](migrations/migrations_test.go) menjaga hal ini — `make test` akan gagal kalau ada migrasi tanpa blok `Down`.

## Menambah modul baru

Pakai [internal/modules/user/](internal/modules/user/) sebagai contoh — isinya CRUD lengkap dengan paginasi, pencarian, dan sorting.

1. Tambahkan tabelnya di [internal/schema/schema.go](internal/schema/schema.go), daftarkan di `All()`, lalu `make migrate-diff name=add_xxx`.
2. Buat folder `internal/modules/xxx/` berisi empat berkas dengan pola yang sama:
   `xxx.module.go`, `xxx.controller.go`, `xxx.service.go`, `xxx.repository.go`, plus folder `dto/`.
3. Rakit modulnya di `app.New` (`xxxModule := xxx.NewModule(db)`) dan simpan di struct `Application`.
4. Tambahkan satu baris `xxxModule.RegisterRoutes(api)` di [internal/app/router.go](internal/app/router.go).

Aturan yang membuat semuanya konsisten:

- Controller **tidak** membentuk response error sendiri — cukup `c.Error(err); return`, `ErrorHandler` yang mengurus sisanya.
- Business logic mengembalikan `apperror.NotFound(...)`, `apperror.Conflict(...)`, dan seterusnya; error tak dikenal otomatis jadi 500 tanpa membocorkan detail internal.
- Repository selalu dipakai lewat interface supaya service mudah di-mock saat test.

## Testing

```bash
make test        # go test ./... -race -cover
```

[internal/app/smoke_test.go](internal/app/smoke_test.go) berisi contoh test HTTP end-to-end tanpa database: modul user dipasang dengan stub `user.Service`, lalu diuji lewat `httptest`. Pakai file itu sebagai pola saat menulis test modul lain.

## Catatan production

- **Jangan deploy apa adanya.** Seluruh endpoint terbuka tanpa autentikasi; pasang lapisan auth lebih dulu.
- Jalankan `./bin/migrate up` sebelum menyalakan aplikasi; `ATLAS_DEV_URL` tidak diperlukan di server.
- Isi `HTTP_ALLOWED_ORIGINS` dengan daftar origin eksplisit, jangan `*`.
- Set `APP_ENV=production` — Gin masuk release mode dan endpoint Swagger dinonaktifkan.
