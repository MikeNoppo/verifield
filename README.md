# Verifield

Pelacakan job order inspeksi lapangan untuk PT Sentra Inspeksi Nusantara
(perusahaan fiktif) — PoC untuk **Case Study 1 — Real-Time Order & Job Tracking**.
Klien melihat status pekerjaannya berubah tanpa memuat ulang halaman; inspektor
memperbarui status dengan satu ketukan yang tetap bekerja di area bersinyal buruk.

Monorepo: `apps/frontend` (Next.js 16 · React 19 · Tailwind v4), `apps/backend`
(Go 1.26 · Gin · GORM · PostgreSQL), `packages/contract` (tipe API ter-generate),
`docs/`, `deploy/`.

## Menjalankan

Satu perintah — seluruh stack (database, migrasi + seed data contoh, backend,
frontend):

```bash
docker compose up -d --build
```

Lalu buka <http://localhost:3000>. Tidak ada login — autentikasi di luar cakupan;
peran dipilih lewat pemilih peran di kanan atas, dan bila satu peran punya lebih
dari satu aktor contoh (dua klien, dua inspektor), dropdown di sampingnya
berpindah antar aktor lewat `?actor=`. Perubahan status yang dilakukan di satu
layar langsung terlihat di layar lain tanpa refresh (SSE).

Pengembangan dengan hot reload (database di Docker, kedua aplikasi langsung di
mesin):

```bash
bun install
cp apps/backend/.env.example apps/backend/.env   # isi SEED_ADMIN_PASSWORD

bun run db:up        # Postgres saja
bun run migrate      # membuat tabel
bun run seed         # mengisi data contoh
bun run dev          # frontend :3000 dan backend :8080 sekaligus
```

Tanpa Docker sama sekali, ganti `bun run db:up` dengan PostgreSQL yang sudah
terpasang, lalu `createdb verifield`.

## Asumsi

Seluruh asumsi bisnis dicatat dan diberi alasan di **Deliverable A**,
[docs/03-deliverable-a.md](docs/03-deliverable-a.md) bagian 2 (A-01…A-08),
ringkasnya: satu order ditangani satu inspektor; satu order terikat satu klien
dan satu lokasi; klien hanya melihat order perusahaannya; operasi dalam satu
zona waktu; perangkat inspektor mampu menyimpan data lokal saat offline; order
aktif bersamaan berjumlah puluhan, bukan puluhan ribu; klien memakai komputer
kantor sedangkan inspektor memakai ponsel; dan koreksi status oleh koordinator
adalah kejadian jarang.

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
| Antrean offline (B-02) | Backend dimatikan, tombol ditekan, backend dinyalakan → laporan menyusul dengan waktu kejadian aslinya utuh |

### Belum / sengaja tidak dikerjakan

Autentikasi, penugasan otomatis, notifikasi keluar (push/surel/SMS), pembayaran,
pelacakan posisi inspektor, unggah foto, dan multi-zona waktu. Alasan
masing-masing ada di [Deliverable A bagian 3](docs/03-deliverable-a.md).

**Case Study 2 (BOM / Work Order / inventori) tidak dikerjakan.** Rubrik
menghitungnya hanya bila kualitas keduanya terjaga, dan satu case yang matang
dinilai lebih tinggi daripada dua yang setengah jadi.

### Di-mock / pengganti sementara

- **Login di-mock** lewat pemilih peran dan header `X-Actor-Id` (isian dari
  `GET /demo/actors`). Otorisasi per peran tetap ditegakkan di server.
- **Compose dan Dockerfile:** berhasil di-build; menjalankan seluruh stack di
  mesin pengembangan tertahan konflik port 3000/8080 dengan project lain —
  bukan cacat compose. Manifest Kubernetes dan pipeline CI belum pernah dijalankan.

## Known Limitations

| Keterbatasan | Konsekuensi | Cara menutupnya |
|---|---|---|
| Identitas tidak diverifikasi (UUID di header) | Siapa pun yang menebak UUID dapat bertindak sebagai pemiliknya. Instance ini tidak boleh menyentuh jaringan publik | JWT + cookie sesi |
| Identitas stream ada di query string | Ikut tercatat di log akses dan riwayat peramban | Cookie sesi — terkirim otomatis oleh EventSource |
| Daftar order diambil `limit=100` lalu disaring di klien | Melewati ~100 order aktif, saringan dan hitungan menjadi salah | Agregasi pindah ke backend. Bergantung pada asumsi A-06 |
| Antrean offline hanya di memori bila `localStorage` diblokir | Laporan hilang bila tab ditutup dalam kondisi itu | IndexedDB, atau Background Sync lewat service worker |
| Belum ada pembatasan laju | Klien nakal bisa membuka banyak stream | Batas koneksi per aktor di reverse proxy |
| Alert `late_update_rejected` belum bisa diselesaikan lewat UI | Koordinator melihat tandanya tetapi belum bisa menutupnya | Kolom `resolved_at` sudah ada; tinggal endpoint dan tombolnya |
| Invarian transaksional belum diuji otomatis | `FOR UPDATE`, idempotensi di bawah beban bersamaan, dan `NOTIFY` transaksional diverifikasi manual | Testcontainers di CI |

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
