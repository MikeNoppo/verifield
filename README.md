# Verifield

Pelacakan job order inspeksi lapangan untuk PT Sentra Inspeksi Nusantara
(perusahaan fiktif). Klien melihat status pekerjaannya berubah tanpa memuat ulang
halaman, dan inspektor punya cara memperbarui status yang tetap bekerja di area
bersinyal buruk.

PoC untuk **Case Study 1 — Real-Time Order & Job Tracking**.

```
apps/frontend/      Next.js 16 · React 19 · Tailwind v4
apps/backend/       Go 1.26 · Gin · GORM · PostgreSQL
packages/contract/  tipe API ter-generate dari anotasi Go
docs/               dokumen bisnis dan desain teknis
deploy/             manifest Kubernetes dan skrip init database
```

Workspace Bun, sehingga seluruh perintah dijalankan dari akar repo. Backend Go
ikut sebagai anggota workspace lewat `package.json` tipis yang membungkus
perintah `go` — satu cara memanggil untuk kedua sisi.

## Dokumen

Kode ini tidak berdiri sendiri. Dua dokumen di bawah adalah bagian yang sama
pentingnya, dan sebagian besar keputusan di kode merujuk balik ke sana.

| Dokumen | Isi |
|---|---|
| [docs/01-business-context.md](docs/01-business-context.md) | Lapisan bisnis: akar masalah, aktor, siklus status, keputusan B-01…B-09, matriks kewenangan, edge case, asumsi A-01…A-08 |
| [docs/02-technical-design.md](docs/02-technical-design.md) | Lapisan teknis: arsitektur, data model, **jawaban atas enam pertanyaan desain wajib**, keterbatasan, alternatif yang ditolak |

## Menjalankan

### Satu perintah

```bash
bun install && bun run stack:up
```

Lalu buka <http://localhost:3000>. Migrasi dan data contoh berjalan otomatis
sebelum aplikasi menyala.

> **Belum pernah diuji.** Berkas compose dan kedua Dockerfile ditulis dengan
> teliti tetapi tidak pernah dijalankan: mesin pengembangan tidak memiliki WSL,
> sehingga engine Linux Docker Desktop tidak bisa menyala. Seluruh pengembangan
> dan verifikasi dilakukan lewat jalur manual di bawah, dengan PostgreSQL native.

### Pengembangan dengan hot reload

Database di Docker, kedua aplikasi langsung di mesin. Semua dari akar repo.

```bash
bun install
cp apps/backend/.env.example apps/backend/.env   # isi SEED_ADMIN_PASSWORD

bun run db:up        # Postgres saja
bun run migrate      # membuat tabel
bun run seed         # mengisi data contoh
bun run dev          # frontend :3000 dan backend :8080 sekaligus
```

Tanpa Docker sama sekali, cukup ganti `bun run db:up` dengan PostgreSQL yang
sudah terpasang, lalu `createdb verifield && createdb verifield_dev`.

Perintah lain: `bun run dev:api`, `bun run dev:web`, `bun run db:reset`,
`bun run stack:logs`.

`verifield_dev` hanya dibutuhkan bila Anda akan meng-generate migrasi baru; ia
dipakai Atlas untuk menghitung diff dan harus kosong.

### Kontrak API

`packages/contract` berisi tipe TypeScript yang **di-generate**, bukan ditulis
tangan:

```
anotasi Go → swag → swagger.json → swagger2openapi → openapi.json
                                  → openapi-typescript → schema.ts
```

Frontend memakai tipe itu untuk seluruh bentuk kawat, termasuk `Status` dan
`Role`. Menambah satu status di backend langsung membuat setiap `switch` di
frontend yang belum menanganinya gagal dikompilasi. `bun run contract:check`
menangkap kontrak yang tertinggal, dan berjalan di CI.

## Mencobanya

Buka <http://localhost:3000> lalu pilih peran. Tidak ada login — autentikasi di
luar cakupan, dan peran diwakili pemilih peran di kanan atas.

**Demonstrasi yang paling menjelaskan sistem ini** membutuhkan dua tab
berdampingan:

1. Tab A: `/lapangan` — buka satu penugasan
2. Tab B: `/klien` — buka order yang sama
3. Tekan tombol besar di tab A → **tab B berubah tanpa refresh**
4. Di tab A, matikan jaringan (DevTools → Network → Offline), tekan tombol lagi →
   muncul *"1 pembaruan menunggu terkirim"*, dan tombol tetap menerima ketukan
5. Nyalakan jaringan → laporan terkirim sendiri, dan riwayat di tab B bertambah
   **satu** entri dengan waktu kejadian saat Anda menekan tombol tadi — bukan
   saat sinyal kembali

Untuk melihat penolakan perubahan bersamaan (B-09), buka `/ops/order/<id>` di dua
tab dan tugaskan inspektor dari keduanya.

## API

Prefix `/api/v1`. Seluruh endpoint job order membutuhkan header `X-Actor-Id`,
yang diisi dari `GET /demo/actors`.

| Method | Path | Keterangan |
|---|---|---|
| GET | `/health` | Status layanan |
| GET | `/demo/actors` | Identitas siap pakai per peran — pengganti login |
| GET | `/inspection-types` | Jenis inspeksi untuk form permintaan |
| GET | `/inspectors` | Inspektor aktif beserta jumlah penugasan berjalan |
| GET | `/orders` | Daftar order. Saring `status`, `company_id`, `inspector_id`, `attention` |
| POST | `/orders` | Buat permintaan inspeksi (klien) |
| GET | `/orders/{id}` | Detail beserta riwayat status |
| GET | `/orders/{id}/events?after_seq=` | Riwayat sejak kursor tertentu |
| POST | `/orders/{id}/events` | Perbarui status dari lapangan (inspektor) — **idempoten** |
| POST | `/orders/{id}/assign` | Tugaskan inspektor (koordinator) — butuh `expected_version` |
| POST | `/orders/{id}/cancel` | Batalkan atau ajukan pembatalan |
| POST | `/orders/{id}/cancellations/{reqId}/decide` | Putuskan permintaan pembatalan |
| POST | `/orders/{id}/corrections` | Koreksi status (koordinator) — wajib beralasan |
| GET | `/stream?last_event_id=` | **SSE** — perubahan order, disaring menurut peran |
| GET · POST | `/users`, `/users/{id}` | CRUD user |

Dokumentasi interaktif: `bun run swagger`, lalu
<http://localhost:8080/swagger/index.html>.

Seluruh response memakai envelope yang sama:

```jsonc
{ "success": true,  "message": "...", "data": {} }
{ "success": false, "message": "...", "code": "CONFLICT" }
{ "success": false, "message": "Validasi gagal", "code": "VALIDATION_ERROR",
  "errors": [{ "field": "reason", "message": "reason wajib diisi" }] }
```

## Status Pengerjaan

### Selesai dan terverifikasi

| Bagian | Bukti |
|---|---|
| Siklus status penuh: buat → tugaskan → berangkat → tiba → mulai → selesai | Diuji end-to-end lewat API dan tampilan |
| Idempotency (B-03) | Tombol ditekan dua kali dengan penanda sama → satu baris riwayat, `duplicate: true` |
| Urutan (B-06) | Laporan mundur ditolak, status tidak berubah, tetap tercatat |
| Laporan terlambat (B-07) | Ditolak setelah status final, dicatat, alert koordinator dibuat |
| Concurrency (B-09) | Versi basi → 409 beserta penjelasannya |
| Pembatalan (B-05) | `In Progress` menjadi permintaan; koordinator menolak → pekerjaan lanjut |
| Kerahasiaan antar klien (A-03) | Order perusahaan lain → 404, bukan 403 |
| Real-time lintas instance | Klien di port 8080 menerima perubahan yang dikirim lewat port 8081 |
| Pemulihan event terlewat | Menyambung ulang dengan kursor `seq` → perubahan yang terlewat dikirim ulang |
| Antrean offline (B-02) | Backend dimatikan, tombol ditekan, backend dinyalakan → laporan menyusul dengan waktu kejadian aslinya utuh (`kejadian 14:57:19`, `diterima 14:57:36`) |

### Sengaja tidak dikerjakan

Autentikasi, penugasan otomatis, notifikasi keluar (push/surel/SMS), pembayaran,
pelacakan posisi inspektor, unggah foto, dan multi-zona waktu. Alasan
masing-masing ada di [01-business-context.md bagian 13](docs/01-business-context.md).

**Case Study 2 (BOM / Work Order / inventori) tidak dikerjakan.** Rubrik
menghitungnya hanya bila kualitas keduanya terjaga, dan satu case yang matang
dinilai lebih tinggi daripada dua yang setengah jadi.

### Ditulis tetapi belum terbukti

`docker-compose.yml`, kedua `Dockerfile`, manifest Kubernetes di `deploy/k8s/`,
dan pipeline CI. Alasannya ada di [Menjalankan](#menjalankan).

## Keterbatasan yang Disadari

Daftar lengkap beserta cara menutupnya ada di
[02-technical-design.md bagian 5](docs/02-technical-design.md). Yang paling
penting:

- **Identitas tidak diverifikasi.** Siapa pun yang menebak sebuah UUID dapat
  bertindak sebagai pemiliknya. Instance ini tidak boleh menyentuh jaringan
  publik.
- **Identitas stream ada di query string,** karena `EventSource` tidak bisa
  memasang header. Ia ikut tercatat di log akses. Cookie sesi menghapus kebutuhan
  ini sepenuhnya.
- **Daftar order diambil `limit=100` lalu disaring di klien.** Bergantung pada
  asumsi A-06 (puluhan order aktif, bukan puluhan ribu). Melewati itu, agregasi
  harus pindah ke backend — bukan sekadar dinaikkan limitnya.
- **Invarian transaksional belum diuji otomatis.** `FOR UPDATE`, idempotency di
  bawah beban bersamaan, dan `NOTIFY` yang transaksional diverifikasi manual
  terhadap Postgres yang berjalan; suite otomatis berjalan tanpa database.

## Pemeriksaan

Seluruhnya dari akar repo:

```bash
bun run lint && bun run typecheck && bun run test
```

Untuk memastikan kontrak API belum tertinggal dari anotasi Go:

```bash
bun run swagger && bun run contract:check
```

## Catatan

Boilerplate backend berasal dari proyek pribadi sebelumnya: struktur modul ala
NestJS, schema terpusat satu berkas ala Prisma, dan pipeline migrasi Atlas →
goose. Seluruh domain job order, lapisan real-time, dan frontend ditulis untuk
tugas ini.
