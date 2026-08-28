# Verifield — Dokumen Desain Teknis

**Case Study 1: Real-Time Order & Job Tracking**
Dokumen ini adalah lapisan teknis. Ia mengambil keputusan bisnis B-01…B-09 dari
[01-business-context.md](01-business-context.md) sebagai batasan, lalu
menjelaskan bagaimana masing-masing ditegakkan di kode.

Setiap keputusan di sini dapat dirujuk kembali ke keputusan bisnis yang
melahirkannya. Bila sebuah mekanisme tidak punya rujukan seperti itu, ia tidak
seharusnya ada.

---

## 1. Arsitektur

```
                      browser
            ┌─────────────────────────┐
            │  Next.js (React 19)     │
            │  /klien /ops /lapangan  │
            └───┬──────────────┬──────┘
        HTTP    │              │  SSE (text/event-stream)
        mutasi  │              │  satu koneksi per layar
                ▼              ▼
        ┌───────────────────────────────┐
        │  Go + Gin  ·  3 instance      │
        │  ┌─────────┐   ┌───────────┐  │
        │  │joborder │   │ realtime  │  │
        │  │ service │   │ hub + SSE │  │
        │  └────┬────┘   └─────▲─────┘  │
        └───────┼──────────────┼────────┘
         tulis  │              │ LISTEN verifield_events
                ▼              │
        ┌───────────────────────────────┐
        │  PostgreSQL                   │
        │  job_status_events (seq)      │
        │  NOTIFY di transaksi yang sama│
        └───────────────────────────────┘
```

**Yang penting dari gambar ini:** tidak ada instance yang berbicara langsung ke
instance lain, dan tidak ada komponen infrastruktur tambahan. Postgres sudah
menjadi dependensi bersama, dan kanal `LISTEN/NOTIFY`-nya yang menghubungkan
seluruh instance.

Sisi klien dibagi dua: Server Component mengambil potret awal supaya layar
pertama langsung terisi, lalu store di browser mengambil alih dan menerapkan
perubahan yang masuk lewat stream.

---

## 2. Data Model

```
companies ──┬── users ──────┬── job_orders ──┬── job_status_events
            │  (client)     │   inspector_id │   seq  ← kursor monotonik
            │               │   created_by   │   occurred_at / received_at
            │               │                │   client_event_id
            │               │                │   accepted / rejection_reason
            │               │                │   is_correction
            │               │                │
            │               │                ├── cancellation_requests
            │               │                └── job_order_alerts
            │               │
inspection_types ───────────┘        reference_counters (JO-2026-0001)
```

### Kolom yang menentukan

| Kolom | Ada karena | Peran |
|---|---|---|
| `job_status_events.seq` | B-01 | `bigserial`. Kursor urutan penerimaan server. Menjadi `id` pesan SSE, sekaligus kursor pemulihan. |
| `occurred_at` / `received_at` | B-02 | Waktu kejadian di lapangan vs waktu terima server. Terpisah karena keduanya sah untuk hal berbeda. |
| `client_event_id` | B-03 | Penanda unik buatan perangkat. Unique index gabungan dengan `job_order_id`. |
| `accepted` + `rejection_reason` | B-06, B-07 | Event yang ditolak tetap tersimpan. Hanya `accepted = true` yang mengubah status. |
| `is_correction` | B-06 | Membedakan koreksi resmi dari transisi biasa. |
| `job_orders.version` | B-09 | Optimistic locking. |
| `job_orders.current_status` | — | **Cache baca-cepat, bukan sumber kebenaran.** Selalu bisa dibangun ulang dari event ber-`accepted = true` dengan `seq` tertinggi. |

### Mengapa riwayat, bukan satu kolom status

Pertanyaan "kapan inspektor tiba di lokasi" dapat dipersengketakan: hasilnya
menjadi dasar sertifikat, klaim asuransi, dan pelepasan pembayaran. Bila status
hanya satu kolom yang ditimpa, jawabannya hilang setiap kali status berubah.

Selain itu, riwayat kejadian adalah satu-satunya struktur yang memenuhi tiga
kebutuhan lain sekaligus: pengiriman ulang event yang terlewat, deteksi event
yang datang terbalik urutannya, dan audit. Ketiganya tidak bisa dibangun di atas
satu kolom yang ditimpa.

**Konsekuensi yang dibayar:** volume data lebih besar, dan membaca status terkini
menuntut penanganan khusus. Itu sebabnya `current_status` ada sebagai cache —
ditulis hanya di transaksi yang sama dengan penyisipan event-nya.

---

## 3. Jawaban atas Pertanyaan Desain

### 3.1 Real-time strategy — mengapa SSE

| Pilihan | Alasan tidak dipilih |
|---|---|
| **Short polling** | Ada jeda antara dua permintaan. Pada jeda itu perubahan tidak terlihat, dan memperkecil jeda berarti memperbanyak permintaan kosong. |
| **Long polling** | Menahan koneksi terbuka seperti SSE, jadi biaya koneksinya setara — tetapi membayar ulang overhead HTTP untuk setiap satu event, dan menyisakan celah antara respons dan permintaan berikutnya. |
| **WebSocket** | Bidirectional, padahal klien hanya membaca. Yang lebih menentukan: reconnect dan pemulihan kursor harus ditulis sendiri dari nol. |

**SSE dipilih karena membawa dua hal yang paling dibutuhkan di sini sebagai
bagian dari protokolnya:**

1. **Penyambungan ulang otomatis.** Browser menanganinya sendiri.
2. **Header `Last-Event-ID`.** Saat menyambung ulang, browser mengirim balik
   `id` pesan terakhir yang diterimanya — tanpa satu baris kode pun.

Poin kedua inilah pembedanya. Kolom `seq` sudah ada karena B-01 menuntut riwayat
kejadian; `Last-Event-ID` berpasangan dengannya persis. Pemulihan event yang
terlewat menjadi konsekuensi dari dua keputusan yang sudah diambil, bukan
mekanisme tambahan.

**Harga yang dibayar, dan bagaimana ditangani:**

- **HTTP/1.1 membatasi enam koneksi per origin.** Klien yang membuka beberapa tab
  bisa menghabiskannya sampai permintaan biasa ikut tertahan. Ditangani dengan
  satu stream per layar (penghitung acuan di `lib/live/store.ts`), bukan satu per
  komponen. Di HTTP/2 batas ini hilang karena koneksinya multiplexed.
- **EventSource tidak bisa memasang header.** Identitas karena itu dikirim lewat
  query. Pada sistem sungguhan hal ini ditangani cookie sesi — yang justru
  terkirim otomatis oleh EventSource. Lihat [Keterbatasan](#5-keterbatasan-yang-disadari).
- **`WriteTimeout` server harus dilepas.** Ia membatasi umur seluruh response,
  sedangkan stream memang dirancang terbuka berjam-jam.

### 3.2 Missed events — koneksi putus lima menit

```
klien terputus (seq terakhir = 42)
   │
   ├─ perubahan terjadi: seq 43, 44, 45
   │
   └─ tersambung kembali
         │
         ├─ browser mengirim Last-Event-ID: 42
         │  (atau klien menyertakan ?last_event_id=42 pada koneksi pertama)
         │
         ├─ server berlangganan LEBIH DULU, baru memutar ulang
         │
         └─ mengirim KEADAAN TERKINI setiap order yang berubah sejak seq 42
```

**Tiga keputusan di dalamnya:**

**Berlangganan sebelum replay, bukan sesudah.** Perubahan yang terjadi di sela
keduanya akan terkirim dua kali. Itu jauh lebih baik daripada tidak terkirim sama
sekali: klien mengabaikan `seq` yang sudah diterapkan, sehingga duplikat tidak
berbahaya — sedangkan celah menghasilkan layar yang diam-diam basi.

**Replay mengirim keadaan sekarang, bukan setiap frame antara.** Klien yang
terputus lima menit hanya perlu tahu di mana ordernya berada sekarang. Riwayat
lengkapnya tetap tersedia lewat `GET /orders/{id}/events?after_seq=`.
*Trade-off:* animasi perpindahan antar status tidak terlihat untuk perubahan yang
terlewat. Itu diterima; yang dibutuhkan klien adalah keadaan, bukan rekaman.

**Kursor juga dikirim klien pada koneksi pertama.** Browser hanya mengirim
`Last-Event-ID` pada penyambungan **ulang**, tidak pada koneksi pertama. Tanpa
`?last_event_id=`, pemulihan setelah halaman dimuat ulang tidak akan pernah
terjadi.

### 3.3 Idempotency & ordering

**Idempotency (B-03).** Inspektor dengan sinyal lemah menekan tombol berulang
kali karena layar tidak segera merespons. Itu bukan kesalahan pengguna melainkan
perilaku manusia yang dapat diperkirakan.

Perangkat membuat `client_event_id` **sebelum kiriman pertama** dan memakainya
kembali untuk setiap pengiriman ulang. Ditegakkan berlapis:

1. Antrean di perangkat menolak penanda yang sudah ada (`lib/offline/queue.ts`).
2. Service memeriksa penanda di dalam transaksi yang sudah memegang
   `SELECT … FOR UPDATE` atas ordernya. Kunci itu yang membuat pemeriksaan
   bebas balapan: seluruh penulisan event untuk satu order berjalan berurutan.
3. Unique index `(job_order_id, client_event_id)` sebagai jaring pengaman.
   Di Postgres, `NULL` tidak pernah bertabrakan di unique index, sehingga event
   buatan sistem bebas berulang tanpa perlu penanda.

Kiriman kedua menghasilkan **200 dengan `duplicate: true`**, bukan error —
perangkat perlu tahu server sudah menerimanya supaya bisa mengeluarkannya dari
antrean.

**Ordering (B-06).** Ada dua urutan yang berbeda, dan mencampurnya adalah sumber
kekacauan:

| Urutan | Ditentukan | Dipakai untuk |
|---|---|---|
| Urutan penerimaan | `seq` | Menentukan status terkini, kursor pemulihan |
| Urutan kejadian | `occurred_at` | Menampilkan timeline kepada pembaca |

Status terkini ditentukan `seq`, bukan `occurred_at`. Empat pembaruan yang tiba
sekaligus setelah sinyal pulih **tidak** membuat status mundur, karena tabel
transisi hanya mengizinkan langkah maju. Pembaruan yang menuntut status mundur
ditolak — tetapi tetap ditulis dengan `accepted = false` dan
`rejection_reason = out_of_order`.

Timeline yang dilihat klien diurutkan `occurred_at`, dengan `seq` sebagai pemecah
seri agar urutannya stabil. Jadi klien melihat urutan yang benar (09.14, 09.20,
11.05) walaupun ketiganya tiba pukul 11.40.

**Jam perangkat yang meleset (B-02).** Waktu kejadian berasal dari jam perangkat
inspektor. `ClampOccurredAt` menolak nilai di luar batas wajar — lebih dari 5
menit di masa depan, lebih dari 7 hari di masa lalu, atau **lebih awal daripada
saat ordernya dibuat** — dan menjatuhkannya ke waktu terima sambil menandai
`occurred_at_adjusted`. Batas ketiga ditambahkan setelah pengujian tampilan
memperlihatkan "tiba di lokasi" muncul sebelum "order diminta".

### 3.4 Concurrency — dua koordinator, satu order

**Perubahan pertama yang menang. Yang kedua ditolak dengan penjelasan (B-09).**

Setiap aksi koordinator membawa `expected_version` — versi yang terlihat di
layarnya saat ia menekan tombol. Service mengunci baris, membandingkan, dan
menolak dengan **409** bila berbeda:

> "Order ini baru saja diubah orang lain, dan sekarang berstatus *ditugaskan*.
> Tampilan Anda sudah diperbarui — silakan periksa lalu ulangi bila masih
> diperlukan."

Koordinator kedua melihat penolakan itu, dan layarnya sudah menampilkan keadaan
terbaru karena pesan real-time tiba lebih dulu.

**Mengapa bukan last-write-wins:** menerima keduanya berarti satu penugasan
hilang tanpa ada yang menyadari, dan dua inspektor berangkat ke lokasi yang sama.
Penolakan yang terlihat jauh lebih baik daripada kehilangan yang tidak terlihat —
terutama pada data yang menjadi dasar dokumen komersial.

**Dua lapis kunci, dan alasannya berbeda:**

- `SELECT … FOR UPDATE` menyerialkan penulisan **di dalam** satu transaksi.
  Ia yang membuat pemeriksaan idempotency bebas balapan.
- Predikat `WHERE version = ?` menegakkan invarian B-09 **di lapisan SQL**.
  Dengan kunci di atas, predikat ini tidak akan pernah meleset — ia ada supaya
  invariannya terbaca di tempat perubahan benar-benar terjadi.

### 3.5 Scaling — tiga pod di belakang load balancer

**Tidak ada instance yang berbicara langsung ke instance lain.**

```
inspektor ──POST──► pod B
                      │  satu transaksi:
                      │  ├─ INSERT job_status_events  (seq 47)
                      │  ├─ UPDATE job_orders          (cache status)
                      │  └─ pg_notify('verifield_events', '47:<order_id>')
                      ▼
                 PostgreSQL
                      │  NOTIFY disiarkan saat COMMIT
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
      pod A         pod B         pod C     ← masing-masing LISTEN
        │
        └─ memuat snapshot, menyaring cakupan, menulis ke stream SSE-nya
                      │
                      ▼
                klien di pod A menerima perubahan yang dikirim lewat pod B
```

**Tiga hal yang membuat ini bekerja:**

**`NOTIFY` bersifat transaksional.** Pesannya baru terkirim saat `COMMIT`,
sehingga mustahil ada pesan real-time untuk perubahan yang ternyata di-rollback.
Jaminan ini tidak didapat kalau penyiaran dilakukan setelah transaksi selesai.

**Tidak perlu sticky session.** Klien boleh mendarat di pod mana pun. Ini yang
membuat `replicas: 3` benar-benar bekerja tanpa session affinity di Ingress.

**Snapshot dimuat sekali per perubahan, bukan sekali per koneksi.** Listener yang
memuatnya, bukan tiap handler SSE. Berapa pun jumlah klien yang terhubung, satu
perubahan tetap satu query.

**Batasnya, dan kapan ia tercapai.** Payload `NOTIFY` dibatasi 8000 byte — di sini
hanya `<seq>:<uuid>`, jauh di bawah batas. Yang lebih dulu terasa adalah setiap
pod memegang satu koneksi database tersendiri untuk `LISTEN` (koneksi yang sedang
`LISTEN` memegang sesinya sepanjang waktu, jadi tidak boleh dipinjam dari pool).
Pada puluhan pod itu masih wajar; pada ratusan, Redis pub/sub atau NATS menjadi
pilihan yang lebih tepat. Perpindahannya menyentuh satu berkas —
`internal/modules/realtime/listener.go`.

**Subscriber lambat di-*drop*, bukan ditunggu.** Menunggu berarti satu klien
lambat bisa menghentikan penyiaran ke semua klien lain, dan pada akhirnya menahan
listener. Aman dilewati karena klien membawa kursor: begitu ia menyusul atau
menyambung ulang, perubahan yang terlewat dikirim ulang. Kehilangan pesan di sini
tidak berarti kehilangan data — hanya menunda kedatangannya.

### 3.6 Auditability

**Satu kolom `status` tidak cukup. `job_status_events` bersifat menambah saja.**

Tabel itu sengaja tidak punya `UpdatedAt` maupun `DeletedAt` — struktur datanya
sendiri yang mencegah baris lama diubah. Koreksi ditulis sebagai baris baru
ber-`is_correction = true`, bukan dengan menimpa.

**Yang tercatat pada setiap entri:** dari status apa ke status apa, oleh siapa
dengan peran apa, kapan kejadiannya di lapangan, kapan diterima sistem, penanda
unik perangkatnya, diterima atau ditolak, alasan penolakannya, dan alasan
tekstual bila ada.

**Setiap perubahan yang terlihat di layar punya `seq`** — termasuk permintaan
pembatalan yang belum diputuskan (`accepted = false`,
`rejection_reason = pending_approval`) dan penolakannya oleh koordinator. Ini
membuat kursor real-time tidak perlu mengenal dua jenis perubahan yang berbeda,
sekaligus membuat jejak auditnya utuh.

**Pembaruan yang ditolak tetap dicatat, dan koordinator diberi tanda.** Klien
membatalkan order ketika inspektor sedang offline; inspektor tidak mengetahuinya,
tetap mengerjakan pemeriksaan, dan laporan `Completed` masuk saat sinyal kembali.
Status tidak berubah karena `Cancelled` bersifat final — tetapi menolak secara
diam-diam adalah kesalahan. Ada pekerjaan nyata yang telah dilakukan seseorang.
Event dicatat `accepted = false`, dan satu baris `job_order_alerts` memberi tanda
agar koordinator dapat menyelesaikan kompensasinya (B-07).

*Alasan mendasarnya:* sistem yang menelan pekerjaan penggunanya tanpa penjelasan
akan ditinggalkan penggunanya. Inspektor akan kembali melapor lewat telepon, dan
masalah awal muncul lagi.

---

## 4. Autentikasi dan Wewenang

Autentikasi dinyatakan di luar cakupan. Yang **tidak** boleh ikut keluar cakupan
adalah wewenangnya: siapa boleh melakukan apa tetap ditegakkan di server.

Identitas datang dari header `X-Actor-Id`, dimuat menjadi user sungguhan oleh
`middleware.RequireActor`. Peran dan perusahaan pemiliknya karena itu tetap
berasal dari database, bukan dari klaim frontend.

**Yang ditegakkan server, apa pun yang dikirim frontend:**

| Aturan | Rujukan |
|---|---|
| Klien hanya melihat order perusahaannya. Order perusahaan lain dijawab **404**, bukan 403 — membedakan keduanya membocorkan keberadaan order milik klien lain | A-03 |
| Inspektor hanya boleh memperbarui order yang ditugaskan kepadanya | — |
| Inspektor tidak berwenang membatalkan, hanya melaporkan kendala | B-04 |
| CS hanya membaca | Bagian 5.4 |
| Koreksi status hanya oleh koordinator, wajib beralasan | B-06 |

**Cara menggantinya nanti:** middleware mengisi `ctxkey.SetActor` dari klaim JWT
alih-alih dari header. Seluruh service tidak berubah, karena mereka sudah
menerima aktor sebagai parameter — bukan menggalinya sendiri dari context.

---

## 5. Keterbatasan yang Disadari

| Keterbatasan | Konsekuensi | Cara menutupnya |
|---|---|---|
| Identitas di header/query, tanpa verifikasi | Siapa pun yang menebak UUID dapat bertindak sebagai pemiliknya | JWT + cookie sesi; cookie justru terkirim otomatis oleh EventSource |
| Identitas stream ada di query string | Ikut tercatat di log akses dan riwayat peramban | Sama seperti di atas — cookie menghapus kebutuhan ini sepenuhnya |
| Daftar order diambil `limit=100`, disaring di klien | Melewati 100 order aktif, saringan dan hitungan menjadi salah | Pindahkan agregasi ke query backend. Bergantung pada asumsi A-06 |
| `docker compose` belum pernah dijalankan | Berkasnya ditulis dengan teliti tetapi belum terbukti | WSL belum terpasang di mesin pengembangan; lihat README |
| Antrean offline hanya di memori bila `localStorage` diblokir | Laporan hilang bila tab ditutup dalam kondisi itu | IndexedDB, atau Background Sync lewat service worker |
| Belum ada pembatasan laju | Klien nakal bisa membuka banyak stream | Batas koneksi per aktor di reverse proxy |
| Alert `late_update_rejected` belum bisa diselesaikan lewat UI | Koordinator melihat tandanya tetapi belum bisa menutupnya | Kolom `resolved_at` sudah ada; tinggal endpoint dan tombolnya |

---

## 6. Alternatif yang Dipertimbangkan dan Ditolak

**Redis pub/sub untuk fan-out.** Ditolak karena menambah satu komponen
infrastruktur untuk masalah yang sudah bisa diselesaikan komponen yang ada.
`NOTIFY` yang transaksional juga memberi jaminan yang tidak dimiliki Redis:
mustahil menyiarkan perubahan yang ternyata di-rollback. Menjadi pilihan tepat
begitu jumlah pod mencapai ratusan.

**Event sourcing penuh.** `job_status_events` sudah menjadi sumber kebenaran,
tetapi `current_status` tetap disimpan sebagai cache. Merekonstruksi status dari
event pada setiap pembacaan akan membuat setiap daftar order menjadi agregasi —
biaya yang tidak sepadan pada skala A-06, dengan imbalan yang sudah didapat dari
riwayat itu sendiri.

**Denormalisasi kolom turunan** (`seq` terakhir, ada tidaknya permintaan
pembatalan, ada tidaknya alert). Dihitung saat dibaca lewat subquery. Menyimpannya
berarti menjaga tiga kolom tetap konsisten di setiap jalur tulis — lebih banyak
tempat untuk salah daripada yang dihemat.

**Server Actions untuk mutasi.** Akan menyembunyikan identitas dari browser, tetapi
menghasilkan dua mekanisme pembaruan yang berbeda: `revalidatePath` untuk mutasi
dan stream untuk perubahan orang lain. Satu jalur — mutasi mengembalikan order,
store menggabungkannya dengan aturan `seq` yang sama seperti pesan stream — lebih
mudah dipertanggungjawabkan.

**Menampilkan posisi inspektor.** Ditolak pada lapisan bisnis (B-08). Posisi
seseorang adalah data pribadi pekerja, dan kebutuhan klien sudah terpenuhi oleh
status.

---

## 7. Bila Diberi Dua Minggu — Tiga Hal Pertama

**1. Autentikasi dan wewenang yang sungguhan.** Bukan karena ia dinilai, tetapi
karena setiap keputusan lain sudah menganggapnya ada. Cookie sesi sekaligus
menghapus identitas dari query string pada stream.

**2. Status `Awaiting Lab Result`.** Dalam praktik nyata, setelah sampel diambil,
pekerjaan tertahan menunggu hasil laboratorium yang bisa memakan hari dan berada
di luar kendali inspektor. Ini status pertama yang akan ditambahkan, dan ia
memperkenalkan aktor eksternal — pertanyaan desain yang benar-benar baru.

**3. Pengujian dengan database sungguhan.** Suite yang ada berjalan tanpa
database, sehingga invarian yang paling penting — `FOR UPDATE`, idempotency di
bawah beban bersamaan, `NOTIFY` yang transaksional — belum pernah diuji secara
otomatis. Semuanya diverifikasi manual terhadap Postgres yang berjalan.
Testcontainers akan mengubah verifikasi itu menjadi bagian dari CI.
