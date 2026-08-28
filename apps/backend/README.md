# Verifield — Backend

REST API pelacakan job order inspeksi lapangan. **Go + [Gin](https://github.com/gin-gonic/gin) + GORM (PostgreSQL)**, disusun memakai struktur modular ala **NestJS** — tiap fitur berdiri sendiri di satu folder berisi controller, service, repository, dan DTO. Schema database terpusat di satu berkas ala **Prisma**.

Kerangka ini berasal dari starter template pribadi; seluruh domain Verifield ditulis untuk case study ini. Rinciannya di [README repositori](../../README.md#basis-kode-yang-dipakai).

Keputusan bisnis yang ditegakkan di sini bernomor `B-01`…`B-10` dan dijelaskan di
[docs/01-business-context.md](../../docs/01-business-context.md); cara masing-masing
ditegakkan di kode ada di [docs/02-technical-design.md](../../docs/02-technical-design.md).

## Menjalankan

Cara termudah adalah lewat compose dari akar repositori (`docker compose up -d --build`),
yang sekaligus menjalankan migrasi dan seed. Untuk menjalankan backend langsung di mesin:

```bash
cp .env.example .env      # sesuaikan DB_*, isi SEED_ADMIN_PASSWORD
docker compose up -d postgres   # dari akar repo — Postgres tidak dipasang natif
go run ./cmd/migrate up   # buat tabel   (~ prisma migrate deploy)
go run ./cmd/seeder       # data contoh  (~ prisma db seed)
go run ./cmd/api          # :8080
```

> `make` tidak terpasang di lingkungan pengembangan ini. Makefile tetap ada sebagai
> dokumentasi perintah, tetapi jalankan perintah `go` di atas secara langsung.

## Peta konsep NestJS → project ini

| NestJS | Di project ini |
|---|---|
| `main.ts` | [cmd/api/main.go](cmd/api/main.go) |
| `app.module.ts` | [internal/app/app.go](internal/app/app.go) |
| `xxx.module.ts` | `internal/modules/xxx/xxx.module.go` |
| `xxx.controller.ts` | `internal/modules/xxx/xxx.controller.go` |
| Repository TypeORM | `internal/modules/xxx/xxx.repository.go` |
| `dto/*.dto.ts` + class-validator | `internal/modules/xxx/dto/*.dto.go` + tag `binding:"..."` |
| `schema.prisma` | [internal/schema/schema.go](internal/schema/schema.go) — semua tabel dalam satu berkas |
| `TransformInterceptor` | [internal/common/response](internal/common/response/response.go) |
| `HttpException` + `ExceptionFilter` | [apperror](internal/common/apperror/apperror.go) + [error_handler.go](internal/common/middleware/error_handler.go) |
| `ValidationPipe` | [httpx.Bind*](internal/common/httpx/bind.go) + [validation](internal/common/validation/validation.go) |

## Struktur folder

```
cmd/
  api/                      entrypoint aplikasi
  migrate/                  penerap migrasi (goose, migrasi ter-embed)
  seeder/                   pembuat admin pertama + data contoh
  atlas-loader/             pencetak DDL dari schema, dipanggil Atlas
internal/
  app/                      perakitan modul + router global
  common/                   infrastruktur lintas modul
    config/ database/ logger/
    apperror/ response/ validation/ httpx/ pagination/
    middleware/             recovery, request logger, CORS, error handler, actor
  schema/                   SCHEMA DATABASE — semua tabel & relasi dalam satu berkas
  modules/
    joborder/               inti domain: order, riwayat status, pembatalan, koreksi
    realtime/               hub SSE + listener LISTEN/NOTIFY
    reference/              data acuan: jenis inspeksi, inspektor, aktor demo
    user/                   CRUD user — juga contoh pola modul
  shared/                   hash, ctxkey
migrations/                 berkas migrasi SQL (ter-embed ke binary)
docs/                       spesifikasi Swagger (hasil generate, ter-commit)
```

## Endpoint

Global prefix `/api/v1`, kecuali `/health` dan `/ready` yang juga tersedia di akar
untuk probe Kubernetes.

| Method | Path | Keterangan |
|---|---|---|
| GET | `/health` | Liveness — sengaja tidak menyentuh dependensi apa pun |
| GET | `/ready` | Readiness — mem-*ping* database, 503 bila belum terjangkau |
| GET | `/demo/actors` | Identitas siap pakai per peran — pengganti login |
| GET | `/inspection-types` · `/inspectors` | Data acuan untuk form dan penugasan |
| GET · POST | `/orders` | Daftar order (tersaring peran) · buat permintaan inspeksi |
| GET | `/orders/{id}` | Detail beserta riwayat status |
| GET | `/orders/{id}/events?after_seq=` | Riwayat sejak kursor tertentu |
| POST | `/orders/{id}/events` | Pembaruan status dari lapangan — **idempoten** (B-03) |
| POST | `/orders/{id}/assign` | Tugaskan inspektor — butuh `expected_version` (B-09) |
| POST | `/orders/{id}/cancel` | Batalkan atau ajukan pembatalan (B-05) |
| POST | `/orders/{id}/cancellations/{reqId}/decide` | Putuskan permintaan pembatalan |
| POST | `/orders/{id}/cancellations/{reqId}/settle` | Putuskan penyelesaian komersial — status tidak berubah (B-10) |
| POST | `/orders/{id}/corrections` | Koreksi status — wajib beralasan (B-06) |
| GET | `/stream?last_event_id=` | **SSE** — perubahan order, disaring menurut peran |
| GET · POST · PATCH · DELETE | `/users`, `/users/{id}` | CRUD user |

> **Tidak ada autentikasi.** Identitas datang dari header `X-Actor-Id` yang diisi dari
> `GET /demo/actors` — autentikasi dinyatakan di luar cakupan oleh soal. **Otorisasinya
> tetap ditegakkan di server:** klien hanya melihat order perusahaannya (404, bukan 403),
> inspektor hanya order yang ditugaskan kepadanya, inspektor tidak berwenang membatalkan
> (B-04), CS hanya membaca, koreksi hanya oleh koordinator. Rancangan penggantinya ada di
> [dokumen teknis bagian 4](../../docs/02-technical-design.md). Jangan menyalakan instance
> ini ke jaringan publik.

Dokumentasi interaktif: `go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/api/main.go -o docs --parseDependency --parseInternal`, lalu buka <http://localhost:8080/swagger/index.html> (nonaktif saat `APP_ENV=production`).

## Bentuk response

Semua endpoint memakai envelope yang sama.

```jsonc
// sukses
{ "success": true, "message": "...", "data": { } }

// list berpaginasi
{ "success": true, "message": "...", "data": [], "meta": { "page": 1, "limit": 10, "total": 42, "total_page": 5 } }

// gagal validasi (422)
{ "success": false, "message": "Validasi gagal", "code": "VALIDATION_ERROR",
  "errors": [{ "field": "email", "message": "email harus berupa alamat email yang valid" }] }
```

Endpoint list menerima `?page=&limit=&search=&sort_by=&sort_dir=`. Kolom `sort_by` difilter whitelist (`SortableColumns` di tiap repository) agar aman dari SQL injection.

## Aturan yang menjaga konsistensi

- **Controller tidak pernah membentuk response error sendiri** — cukup `c.Error(err); return`, `ErrorHandler` yang mengurus sisanya. Service mengembalikan `apperror.NotFound(...)`, `apperror.Conflict(...)`, dan seterusnya; error tak dikenal jadi 500 tanpa membocorkan detail internal.
- **Aktor diserahkan eksplisit sebagai parameter** ke service, bukan digali dari context di dalamnya. Itulah yang membuat aturan bisnis bisa diuji tanpa gin, dan yang membuat penggantian header `X-Actor-Id` dengan JWT nanti tidak menyentuh satu pun service.
- **Repository selalu dipakai lewat interface**, sehingga service bisa diuji dengan fake — lihat [cancellation_test.go](internal/modules/joborder/cancellation_test.go).
- **`current_status` adalah cache, bukan sumber kebenaran** (B-01). Ia hanya boleh ditulis di transaksi yang sama dengan penyisipan event ber-`accepted = true`.

## Schema dan migrasi

Seluruh schema ada di **satu berkas**: [internal/schema/schema.go](internal/schema/schema.go) — padanan `schema.prisma`. Struct di berkas itu adalah **sumber kebenaran**; perubahannya diterjemahkan menjadi berkas SQL bernomor.

Pembagian tugasnya: **Atlas meng-generate** SQL dari `schema.go`, **goose menerapkannya**.

| Prisma | Di sini |
|---|---|
| `prisma migrate dev --name add_x` | `atlas migrate diff add_x --env gorm` |
| `prisma migrate deploy` | `go run ./cmd/migrate up` |
| `prisma db seed` | `go run ./cmd/seeder` |

Atlas **tidak pernah menyentuh database production** — ia hanya membaca `schema.go`, memutar ulang berkas di [migrations/](migrations/) pada sebuah dev database kosong, lalu menulis selisihnya sebagai berkas baru. Yang menyentuh production hanya `cmd/migrate`, dan berkas migrasinya **ter-embed di dalam binary** ([migrations/embed.go](migrations/embed.go)) — server tidak perlu Atlas, tidak perlu folder `migrations/`, dan tidak perlu Go toolchain.

Migrasi sengaja **tidak** dijalankan otomatis saat aplikasi start, supaya beberapa instance yang menyala bersamaan tidak saling berebut mengubah schema. Di Kubernetes ia berjalan sebagai `Job`, bukan `initContainer` — alasannya di [deploy/k8s/README.md](../../deploy/k8s/README.md).

Atlas tidak terpasang di lingkungan ini. Untuk melihat DDL tanpa database: `go run ./cmd/atlas-loader`.

## Testing

Seluruh suite berjalan **tanpa database**:

```bash
go build ./... && go vet ./... && gofmt -l .   # gofmt -l: keluaran kosong = rapi
go test ./...
```

| Berkas | Yang dijaga |
|---|---|
| [transition_test.go](internal/modules/joborder/transition_test.go) | Tabel transisi hanya maju; tidak ada jalan keluar dari status final; batas kewajaran `occurred_at` (B-02, B-06) |
| [visibility_test.go](internal/modules/joborder/visibility_test.go) | Batas baca per peran, dipakai jalur baca maupun jalur siaran (A-03) |
| [cancellation_test.go](internal/modules/joborder/cancellation_test.go) | Status final tidak dapat dibuka kembali, permintaan yang tersusul berpindah menunggu penyelesaian, dan hasilnya tercatat (B-10) |
| [hub_test.go](internal/modules/realtime/hub_test.go) | Fan-out SSE dan pemulihan kursor |
| [smoke_test.go](internal/app/smoke_test.go) | Rantai middleware, envelope validasi, password tidak pernah bocor |
| [migrations_test.go](migrations/migrations_test.go) | Setiap migrasi punya blok `-- +goose Down` yang tidak kosong |

Yang **belum** diuji otomatis: perilaku SQL-nya sendiri — `SELECT … FOR UPDATE`, idempotensi di bawah beban bersamaan, dan sifat transaksional `NOTIFY`. Ketiganya diverifikasi manual terhadap Postgres yang berjalan; Testcontainers di CI adalah langkah berikutnya.

## Menambah modul baru

Pakai [internal/modules/user/](internal/modules/user/) sebagai contoh — CRUD lengkap dengan paginasi, pencarian, dan sorting.

1. Tambahkan tabelnya di [internal/schema/schema.go](internal/schema/schema.go), daftarkan di `All()`, lalu generate migrasinya.
2. Buat folder `internal/modules/xxx/` berisi `xxx.module.go`, `xxx.controller.go`, `xxx.service.go`, `xxx.repository.go`, plus folder `dto/`.
3. Rakit modulnya di `app.New` ([internal/app/app.go](internal/app/app.go)).
4. Tambahkan satu baris `xxxModule.RegisterRoutes(api)` di [internal/app/router.go](internal/app/router.go).
