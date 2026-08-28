# Verifield — Panduan Menguji Seluruh Alur

Panduan ini menjalankan PoC dari awal sampai akhir: setiap alur bisnis, setiap
keputusan B-01…B-09, dan jawaban atas pertanyaan desain wajib dapat dibuktikan
lewat langkah-langkah di bawah. Nomor keputusan merujuk ke
[01-business-context.md](01-business-context.md).

---

## 0. Persiapan

```bash
docker compose up -d --build
```

| Hal | Nilai |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:8080/api/v1 |
| Swagger | `bun run swagger` lalu http://localhost:8080/swagger/index.html |
| Login | Tidak ada. Peran dipilih di kanan atas; aktor dipilih lewat dropdown `?actor=` |

**Aktor contoh** (id berubah tiap kali database di-reset — salin dari
`GET /demo/actors` atau lihat dropdown):

| Peran | Nama | Catatan |
|---|---|---|
| Koordinator | Administrator | melihat & mengatur semua |
| Klien | Budi Santoso · PT Samudra Ekspor Nusantara (SEN) | 2 order |
| Klien | Dewi Lestari · PT Bara Timur Perkasa (BTP) | 2 order |
| Inspektor | Rina Amelia | tersibuk → default dropdown inspektor |
| Inspektor | Joko Prasetyo | |
| CS | Sari Wijaya | hanya baca |

**Data contoh** (order diberi nomor urut, jadi `JO-2026-000X` bisa bergeser —
kenali lewat objeknya):

| Objek | Pemilik | Status awal | Inspektor |
|---|---|---|---|
| Batu bara 5.000 MT | SEN | `Completed` | Joko |
| Stockpile batu bara blok C | BTP | `In Progress` | Rina |
| 40 kontainer CPO | SEN | `Requested` | — |
| Tangki timbun T-401 | BTP | `Cancelled` + laporan telat ditolak (B-07) | Joko |

**Mengembalikan data contoh** setelah dicoba-coba:

```bash
docker compose down -v && docker compose up -d
```

---

## 1. Alur Klien

### 1.1 Memantau daftar order (F-01)

1. Buka `http://localhost:3000/klien`.
2. Sebagai Budi (SEN), daftar menampilkan order SEN: satu `Selesai`, satu `Diminta`.
3. Klik saringan **Berjalan / Selesai / Bermasalah** — daftar tersaring.
4. **Hasil:** tidak ada satu pun order BTP (JO "Stockpile" dan "Tangki") di layar.

### 1.2 Kerahasiaan antar klien (A-03)

1. Salin id order SEN dari `/ops` (atau dari URL detail).
2. Buka `http://localhost:3000/klien/order/<id-order-SEN>` sebagai Budi → halaman terbuka (200).
3. Pindah ke Dewi lewat dropdown aktor (`?actor=`) lalu buka URL yang sama → **halaman 404**.
4. **Hasil:** backend menjawab "tidak ditemukan", bukan "tidak boleh" — keberadaan
   order klien lain tidak bocor.

### 1.3 Membuat permintaan baru

1. Buka `/klien/permintaan-baru`.
2. **Uji validasi tanggal:** coba pilih tanggal kemarin pada "Jadwal yang diminta" —
   kalender tidak mengizinkan (batas bawah = sekarang).
3. Isi formulir (jenis inspeksi dropdown menampilkan **nama**, bukan id), kirim.
4. **Hasil:** layar "Permintaan diterima" dengan nomor `JO-…`; buka "Lihat order ini" →
   status `Diminta`; kembali ke daftar, order baru sudah ada (tanpa refresh manual).

### 1.4 Timeline detail (F-02, B-01)

1. Buka detail order "Batu bara 5.000 MT" (Completed).
2. Timeline menampilkan seluruh tahap berurutan dengan waktu kejadian, aktor, dan peran.
3. **Hasil:** riwayat adalah baris-baris kejadian (append-only), bukan satu nilai yang ditimpa.

### 1.5 Pembatalan — matriks kewenangan (F-05, B-05)

| Dari status | Yang terjadi |
|---|---|
| `Requested` (order baru §1.3) | Tombol "Batalkan Order" → langsung batal, tanpa biaya |
| `On The Way` / `On Site` | Langsung batal + peringatan **biaya kunjungan** (inspektor sudah dikerahkan) |
| `In Progress` | Tombol berubah jadi "Ajukan Pembatalan" → status **tidak** berubah, muncul penjelasan "diteruskan ke koordinator" |
| `Completed` / `Failed` / `Cancelled` | Tombol nonaktif |

Cara memicu kasus `On The Way`: tugaskan inspektor ke order baru, buka `/lapangan`
sebagai inspektor itu, tekan **Berangkat**, lalu kembali ke `/klien` dan batalkan.

1. Di sisi Dewi, buka "Stockpile batu bara blok C" (In Progress) → "Ajukan Pembatalan",
   isi alasan, kirim.
2. **Hasil:** pekerjaan terus berjalan; di `/ops` order tampil dengan tanda permintaan
   pembatalan (§2.4).

---

## 2. Alur Koordinator

### 2.1 Penugasan (F-03)

1. Buka `/ops` — seluruh order tampil dalam satu layar; filter perhatian di kanan
   (`?a=penugasan` dan seterusnya).
2. Buka order "40 kontainer CPO" (`Requested`), klik **Tugaskan**.
3. Dropdown inspektor menampilkan **nama + jumlah penugasan berjalan**, urut dari
   yang paling sepi. Pilih salah satu.
4. **Hasil:** status → `Assigned`; klien (Budi) melihatnya berubah tanpa refresh;
   order muncul di `/lapangan` inspektor yang dipilih (cek lewat dropdown `?actor=`).

### 2.2 Concurrency — dua koordinator (B-09)

1. Buka detail order `Requested` yang sama di **dua tab**.
2. Di tab 1 tugaskan **Rina**; di tab 2 tugaskan **Joko**.
3. **Hasil:** tab 2 ditolak dengan pesan manusiawi ("Order ini baru saja diubah…"),
   tampilannya sudah memperbarui diri; hanya satu inspektor yang tercatat.

### 2.3 Koreksi status (F-06, B-06)

1. Buka detail order "Batu bara 5.000 MT" (`Completed`) di `/ops`, klik **Koreksi Status**.
2. Kembalikan ke `On Site` dengan alasan ≥ 8 karakter.
3. **Hasil:** entri lama tetap utuh; koreksi muncul sebagai entri baru di timeline
   dengan alasan; status berubah.

### 2.4 Memutuskan permintaan pembatalan (B-05)

1. Setelah §1.5, buka order "Stockpile" di `/ops` — ada tanda permintaan pembatalan.
2. Klik **Tolak** → **Hasil:** pekerjaan lanjut sebagai `In Progress`, Dewi melihat
   pembatalannya ditolak.
3. Ulangi dengan order lain → **Setujui** → status `Cancelled`, biaya ditentukan koordinator.

### 2.5 Alert laporan terlambat (B-07)

1. Di `/ops`, order "Tangki timbun T-401" tampil dengan tanda perhatian.
2. Buka detailnya: timeline menampilkan entri `Completed` berlabel **ditolak —
   datang setelah status final**, dan ada alert bahwa kompensasi inspektor perlu
   diselesaikan.
3. **Hasil:** pekerjaan nyata tidak ditelan sistem; ia dicatat dan dilaporkan.

---

## 3. Alur Inspektor

### 3.1 Tugas hari ini

1. Buka `/lapangan` (default: Rina, inspektor tersibuk).
2. **Hasil:** hanya penugasan Rina yang tampil ("Stockpile", In Progress). Order Joko
   tidak muncul. Ganti inspektor lewat dropdown `?actor=` → daftar ikut berganti.

### 3.2 Batas baca inspektor

1. Salin id order SEN (mis. "Batu bara 5.000 MT") dari `/ops`.
2. Buka `/lapangan/order/<id>` → **halaman 404**. Lewat API: `GET /orders/{id}`
   dengan `X-Actor-Id` Rina → `404`; saringan `inspector_id` lain juga dipaksa server.
3. **Hasil:** inspektor tidak dapat membaca order yang bukan tugasnya.

### 3.3 Satu ketukan kontekstual (F-04) + real-time lintas layar

1. **Tab A:** `/lapangan/order/<id-stockpile>` (Rina). **Tab B:** `/klien` (Dewi).
2. Di tab A tekan tombol besar kontekstual (saat ini **Selesai**).
3. **Hasil:** tab B berubah **tanpa refresh** — status dan timeline bertambah satu
   entri dengan waktu kejadian saat ketukan.

### 3.4 Offline — antrean di perangkat (B-02)

1. Di tab A, DevTools → Network → **Offline**. Tekan tombol berikutnya (mis. dari
   `On Site` → `Mulai`).
2. **Hasil:** muncul "… tercatat — tersimpan di perangkat, menunggu terkirim";
   tombol tetap menerima ketukan.
3. Nyalakan kembali jaringan. **Hasil:** laporan terkirim sendiri; di tab B riwayat
   bertambah **satu** entri, dengan waktu kejadian saat tombol ditekan — bukan saat
   sinyal kembali.

### 3.5 Idempotensi ketukan ganda (B-03)

1. Dalam keadaan offline, tekan tombol yang sama berulang kali (atau dua kirim dengan
   `client_event_id` yang sama lewat API — lihat §5).
2. **Hasil:** setelah online, hanya **satu** baris riwayat; kiriman kedua dijawab
   `duplicate: true`, bukan error.

---

## 4. Real-Time & Pemulihan

### 4.1 Indikator koneksi

1. Perhatikan indikator di kanan atas (tersambung / menyambung ulang / terputus).
2. Hentikan backend: `docker compose stop backend`, amati indikator → menyambung ulang.
3. Nyalakan: `docker compose start backend` → indikator pulih, dan perubahan yang
   terjadi selama jeda **dikirim ulang otomatis** (kursor `seq` + `Last-Event-ID`).

### 4.2 Stream SSE mentah

```bash
curl -N "http://localhost:8080/api/v1/stream?last_event_id=0&actor_id=<id-aktor>"
```

Biarkan terbuka, lalu ubah status lewat tab lain → pesan `event: order` dengan
`id: <seq>` masuk ke terminal. Matikan `curl`, ubah status sekali lagi, sambungkan
ulang dengan `last_event_id=<seq-terakhir>` → perubahan yang terlewat terkirim ulang.

---

## 5. Melalui API Langsung (curl / Postman)

Semua endpoint job order membutuhkan header `X-Actor-Id` (ambil dari `/demo/actors`).
Respons memakai envelope `{success, message, data}`.

```bash
BASE=http://localhost:8080/api/v1
ACTOR=<id-aktor>

# Daftar aktor, jenis inspeksi, inspektor
curl -s $BASE/demo/actors | jq
curl -s $BASE/inspection-types | jq
curl -s $BASE/inspectors | jq

# Daftar order sebagai aktor tertentu (saringan perusahaan/inspektor dipaksa server)
curl -s -H "X-Actor-Id: $ACTOR" "$BASE/orders?limit=100" | jq

# Idempotensi (B-03): penanda sama dikirim dua kali
# (occurred_at boleh dikosongkan; server memakai waktu terima sebagai cadangan)
curl -s -X POST -H "X-Actor-Id: $ACTOR" -H "Content-Type: application/json" \
  -d '{"to_status":"completed","client_event_id":"uji-idempoten-1"}' \
  "$BASE/orders/<id-order-milik-aktor>/events" | jq
# kirim ulang persis sama → { "duplicate": true, "accepted": true }

# Urutan (B-06): transisi mundur ditolak tetapi tetap tercatat
curl -s -X POST -H "X-Actor-Id: $ACTOR" -H "Content-Type: application/json" \
  -d '{"to_status":"assigned","client_event_id":"uji-mundur-1"}' \
  "$BASE/orders/<id-order-in-progress>/events" | jq
# → accepted=false, rejection_reason="out_of_order"

# Concurrency (B-09): versi basi
curl -s -X POST -H "X-Actor-Id: $ACTOR" -H "Content-Type: application/json" \
  -d '{"inspector_id":"<id>","expected_version":0}' \
  "$BASE/orders/<id>/assign" | jq
# → 409 dengan penjelasan

# Laporan terlambat (B-07): kirim completed ke order yang sudah Cancelled
# → accepted=false, rejection_reason="late_after_final", alert dibuat (cek /ops)

# Riwayat sejak kursor tertentu
curl -s -H "X-Actor-Id: $ACTOR" "$BASE/orders/<id>/events?after_seq=0" | jq
```

---

## 6. Matriks Pemeriksaan Cepat

| Keputusan | Bukti | Skenario |
|---|---|---|
| B-01 riwayat append-only | Timeline lengkap, koreksi = baris baru | §1.4, §2.3 |
| B-02 waktu kejadian vs terima | Waktu di timeline = saat ketukan, bukan saat sinyal pulih | §3.4 |
| B-03 idempotensi | Satu baris riwayat, `duplicate: true` | §3.5, §5 |
| B-04 inspektor tak bisa membatalkan | Tidak ada tombol batal di `/lapangan`; `Failed` ber-alasan | §3 |
| B-05 pembatalan jadi permintaan | "Ajukan Pembatalan" + keputusan koordinator | §1.5, §2.4 |
| B-06 hanya maju, koreksi resmi | Transisi mundur ditolak; koreksi beralasan | §5, §2.3 |
| B-07 terlambat tetap dicatat | Entri ditolak + alert di `/ops` | §2.5, §5 |
| B-09 perubahan pertama menang | 409 + penjelasan di tab kedua | §2.2 |
| A-03 kerahasiaan antar klien | Order perusahaan lain → 404 | §1.2 |
| Batas baca inspektor | Order bukan tugasnya → 404, daftar tersaring server | §3.2 |
| SSE + missed events | Tab berubah tanpa refresh; reconnect mengirim ulang | §3.3, §4 |
