# **Technical Assignment — Fullstack Developer** 

**Posisi:** Fullstack Developer **Stack:** Frontend (React / Vue) + Backend (Golang) **Durasi:** 24 jam sejak dokumen ini diterima 



### **Yang kami nilai** 

|**Aspek**|**Apa yang kami lihat**|
|---|---|
|**Business process**<br>**understanding**|Memetakan alur bisnis, aktor, dan_edge case_sebelum menulis<br>kode|
|**Kemampuan teknis inti**|Kualitas struktur kode, desain API, dan data model pada<br>Frontend(React/Vue)& Backend(Golang).|
|**Infrastructure awareness**<br>_(huge plus)_|Pemahaman Docker, Kubernetes, dan CI/CD — konseptual<br>maupunpraktikal.|



### **Yang TIDAK kami nilai** 

- Kelengkapan fitur. PoC seadanya yang dipikirkan matang **jauh lebih bernilai** daripada aplikasi lengkap tanpa alasan di balik desainnya. 

- Kecantikan UI. Rapi dan intuitif sudah cukup; tidak perlu design system atau animasi. 

- Test coverage 100%, authentication/RBAC lengkap, atau production hardening. 

- Jumlah baris kode. 

**Prinsip utama:** Jika Anda harus memilih antara "menyelesaikan satu fitur lagi" atau "merapikan dokumen dan menjelaskan alasan keputusan Anda" — **pilih yang kedua.** 



## **2. Aturan Pengerjaan** 

1. **Kerjakan 2 case study** di bawah ini 

2. **Anda bebas berasumsi.** Deskripsi case sengaja dibuat tidak lengkap — sama seperti _requirement_ di dunia nyata. Jika ada informasi bisnis yang kurang, buat asumsi sendiri, **asalkan asumsi tersebut dicatat dan dikomunikasikan secara eksplisit** di dokumen Anda. Kualitas asumsi Anda adalah salah satu hal yang paling kami perhatikan. 

3. **Boleh memakai library/framework apa pun** selama stack inti tetap React/Vue + Golang. 

4. **Commit history yang rapih.** 



## **3. Case Study** 

Kerjakan ke duanya sesuai dengan batas waktu 24 jam setelah test ini diterima 



### **CASE STUDY 1 — Fitur "Real-Time Order & Job Tracking"** 

#### **Konteks Bisnis** 

Sebuah platform layanan operasional (jasa lapangan) saat ini kewalahan menangani telepon dan chat dari klien yang menanyakan hal yang sama berulang kali: _"Pekerjaan saya sudah sampai mana?"_ 

Tim Customer Service menghabiskan sebagian besar waktunya hanya untuk menjawab pertanyaan status. Manajemen ingin membangun fitur **tracking** agar klien bisa melihat status pekerjaan mereka **secara langsung tanpa me-refresh halaman** , sehingga beban CS berkurang secara signifikan. 

Status pekerjaan (contoh, silakan sesuaikan dengan asumsi Anda): `To Do` → `In Progress` → `Done` , dengan kemungkinan `Cancelled` dan status lain yang Anda anggap perlu. 

#### **Aktor yang Terlibat** 

- **Klien** — memesan layanan, memantau status, dapat membatalkan pesanan. 

- **Admin / Dispatcher** — menugaskan pekerjaan, memantau seluruh order. 

- **Field Officer / Teknisi** — mengeksekusi pekerjaan di lapangan dan memperbarui status, seringkali dari perangkat mobile dengan koneksi tidak stabil. 

#### **Tantangan Bisnis yang Harus Anda Pikirkan** 

1. **Sinkronisasi data yang sering terlambat.** Update dari lapangan bisa datang terlambat, keluar urutan ( _out-of-order_ ), atau terkirim ganda saat teknisi menekan tombol berulang kali. 

2. **Pembatalan mendadak.** Klien membatalkan pesanan saat teknisi sudah dalam perjalanan, atau bahkan saat pekerjaan sudah `In Progress` . Apa yang terjadi? Siapa yang berhak membatalkan? Sampai titik mana pembatalan masih diizinkan? 

3. **Antarmuka yang intuitif.** Klien non-teknis harus langsung paham status pekerjaannya tanpa perlu penjelasan. 

#### **Pertanyaan Desain yang Wajib Dijawab di Dokumen** 

Ini adalah inti dari case study ini — lebih penting daripada kodenya. 

- **Real-time strategy:** Anda memilih _short polling_ , _long polling_ , _Server-Sent Events (SSE)_ , atau _WebSocket_ ? Jelaskan mengapa anda memilihnya 

- **Missed events:** Apa yang terjadi jika koneksi klien terputus 5 menit lalu tersambung kembali? Bagaimana ia mendapatkan perubahan yang terlewat? 

- **Idempotency & ordering:** Bagaimana Anda memastikan event ganda tidak menghasilkan status ganda, dan event yang datang terbalik urutannya tidak membuat status "mundur"? 

- **Concurrency:** Dua admin mengubah status order yang sama secara bersamaan. Siapa yang menang, dan bagaimana pengguna lain tahu? 

- **Scaling:** Jika backend Anda berjalan di 3 pod/instance di belakang load balancer, bagaimana event dari instance A sampai ke klien yang terhubung ke instance B? 

- **Auditability:** Bagaimana Anda menyimpan riwayat perubahan status? (Apakah cukup satu kolom `status` , atau perlu tabel event/history?) 

#### **Scope Minimum PoC** 

Cukup ini saja — jangan lebih: 

- **Backend (Golang):** 

   - Endpoint untuk membuat order (boleh seeded dummy data). 

   - Endpoint untuk mengubah status order. 

   - Endpoint untuk membatalkan order, dengan validasi transisi status. 

   - Satu mekanisme real-time (sesuai pilihan Anda) yang mengirim perubahan ke frontend. 

   - _Persistence_ bebas: in-memory, SQLite, atau PostgreSQL. In-memory dapat diterima selama Anda menjelaskan konsekuensinya. 

- **Frontend (React/Vue):** 

   - Halaman daftar order yang statusnya berubah **tanpa refresh manual** . 

   - Halaman/panel detail order dengan _timeline_ riwayat status. 

   - Aksi "Cancel Order" beserta penanganan error/penolakannya. 

   - Indikator koneksi (connected / reconnecting) — sederhana saja, tapi menunjukkan Anda memikirkan kegagalan. 

#### **Konteks Bisnis** 

Sebuah perusahaan ritel memproduksi sendiri sebagian barang yang mereka jual. Saat ini semuanya dicatat di spreadsheet manual: resep produk, perintah produksi, dan stok gudang. Akibatnya sering terjadi: 

- Barang **terjual di dua channel sekaligus** padahal stoknya hanya satu ( _oversell_ ). 

- Tim produksi mengeluarkan bahan baku yang ternyata sudah dialokasikan untuk _work order_ lain. 

- Stok di sistem tidak pernah cocok dengan stok fisik. 

Mereka ingin beralih ke satu sistem terpusat yang mengelola **Bill of Materials (BOM)** , **Work Orders** , dan **pemotongan stok inventori** secara akurat untuk mendukung penjualan **omnichannel** (toko offline, webstore, dan marketplace). 

#### **Alur Bisnis Inti (garis besar)** 

```
Sales Order (dari berbagai channel)
        │
        ▼
  Cek ketersediaan stok barang jadi
        │
        ├── cukup ──► Alokasi & kirim
        │
        └── kurang ─► Buat Work Order
                            │
                            ▼
                   BOM explosion → hitung kebutuhan bahan baku
                            │
                            ▼
                   Reserve / issue bahan baku
                            │
                            ▼
                   Produksi selesai → stok barang jadi bertambah,
                                       bahan baku berkurang
```

#### **Tantangan Bisnis yang Harus Anda Pikirkan** 

1. **Race condition & alokasi tumpang tindih.** Dua order (atau dua _work order_ ) memperebutkan stok bahan baku yang sama pada saat bersamaan. Bagaimana Anda mencegah stok "terjual dua kali"? 

2. **Desain relasi data.** Bagaimana memodelkan hubungan dari bahan baku → BOM → barang jadi, termasuk kemungkinan komponen bertingkat ( _sub-assembly_ )? 

3. **Antarmuka untuk staf gudang/produksi.** Penggunanya bukan orang IT; sering bekerja sambil berdiri, terburu-buru, dan tidak akan membaca tooltip. 

#### **Pertanyaan Desain yang Wajib Dijawab di Dokumen** 

- **Konsep stok:** Bagaimana Anda membedakan `on hand` , `reserved/allocated` , dan `available` ? Mengapa pembedaan ini penting untuk mencegah oversell? 

- **Race condition:** Mekanisme apa yang Anda pilih — database transaction + `SELECT ... FOR UPDATE` , _optimistic locking_ dengan kolom versi, _unique constraint_ , antrian/queue, atau kombinasi? Jelaskan trade-off-nya (throughput, deadlock risk, kompleksitas). 

- **BOM versioning:** Resep produk berubah bulan depan. Bagaimana memastikan _work order_ lama tetap merefleksikan resep saat ia dibuat, dan laporan historis tidak ikut berubah? 

- **Unit of Measure:** Bahan baku dibeli dalam meter/kilogram, tapi dipakai dalam sentimeter/gram. Bagaimana Anda menanganinya? 

- **Skenario tidak ideal:** Produksi selesai sebagian ( _partial completion_ ), ada bahan rusak/ _scrap_ , work order dibatalkan di tengah jalan, atau hasil _stock opname_ berbeda dari sistem. Bagaimana sistem Anda merespons? 

- **Sumber kebenaran omnichannel:** Jika marketplace punya cache stok sendiri, bagaimana strategi sinkronisasinya? (Cukup konseptual, tidak perlu diimplementasi.) 

#### **Scope Minimum PoC** 

Cukup ini saja — jangan lebih: 

- **Backend (Golang):** 

   - CRUD sederhana untuk Material dan Product. 

   - Endpoint membuat BOM untuk sebuah produk. 

   - Endpoint membuat Work Order (yang meng- _explode_ BOM dan mereservasi bahan baku). 

   - Endpoint menyelesaikan Work Order (pemotongan stok bahan baku + penambahan stok barang jadi) — **operasi ini harus atomik** , dan tunjukkan di kode bagaimana Anda menjaganya. 

   - _Persistence_ sebaiknya menggunakan database relasional 

      - (PostgreSQL/MySQL/SQLite) agar mekanisme _locking_ Anda terlihat. 

- **Frontend (React/Vue):** 

   - Halaman daftar stok (menampilkan on hand / reserved / available). 

   - Form membuat Work Order dengan preview kebutuhan bahan baku hasil BOM explosion. 

   - Aksi "Selesaikan Work Order" beserta penanganan error saat stok tidak mencukupi. 



## **4. Deliverables** 

Ada **dua** deliverable. Keduanya wajib. Deliverable A memiliki bobot penilaian yang sama besar dengan Deliverable B — mohon jangan mengerjakannya di 15 menit terakhir. 

### **A. Dokumen atau Slide Deck** 

Format bebas (PDF, Google Slides, Notion, atau Markdown di dalam repo). Panjang ideal: **8–12 slide** atau **3–5 halaman** . Singkat, padat, terstruktur. 

Wajib memuat: 

1. **Problem understanding** — jelaskan ulang masalahnya dengan kalimat Anda sendiri. Menurut Anda, apa akar masalah sebenarnya bagi bisnis? 

2. **Assumptions** — daftar eksplisit semua asumsi yang Anda ambil. Contoh: _"Saya berasumsi satu order hanya memiliki satu teknisi, karena…"_ 

3. **Scope: In / Out** — apa yang Anda kerjakan, apa yang sengaja tidak dikerjakan, dan **mengapa** . 

4. **Architecture diagram** — sederhana saja. Kotak dan panah sudah cukup. 5. **Data model / ERD sederhana** — entitas, relasi, dan field kunci. 6. **Process flow** — alur bisnis utama (boleh flowchart atau sequence diagram). 7. **Edge cases & mitigasi** — bagian ini kami baca paling teliti. Sajikan dalam bentuk tabel: _skenario → risiko → cara sistem menanganinya_ . 

8. **Trade-off & alternatif yang ditolak** — apa opsi lain yang Anda pertimbangkan, dan mengapa tidak dipilih. 

9. **Jawaban atas "Pertanyaan Desain yang Wajib Dijawab"** pada case study yang Anda pilih. 

10. **What's next** — jika Anda diberi 2 minggu penuh, apa 3 hal pertama yang akan Anda kerjakan? 

### **B. Repository GitHub (Dummy Code / PoC)** 

Repository publik (atau privat dengan akses untuk [username reviewer]). 

Wajib memuat: 

- **Frontend** — React atau Vue. 

- **Backend API** — Golang. 

- **README.md** yang berisi: 

   - Cara menjalankan project (idealnya cukup satu perintah). 

   - Daftar asumsi (boleh merujuk ke Deliverable A). 

   - **Status pengerjaan** — apa yang sudah selesai, apa yang belum, apa yang di- _mock_ . 

   - **Known limitations** — batasan yang Anda sadari. Menuliskan keterbatasan yang Anda sadari **menaikkan** nilai Anda, bukan menurunkannya. 

   - Daftar endpoint API (tabel sederhana sudah cukup). 

- **Seed / dummy data** agar reviewer bisa langsung mencoba tanpa input manual. 

Boleh dalam satu repo (monorepo dengan folder `/frontend` dan `/backend` ) atau dua repo terpisah. 



## **5. Huge Plus — Infrastructure & Delivery** 

Bagian ini **opsional** , tetapi bobotnya besar dan menjadi pembeda utama antar kandidat. 

**Item** 

#### **Bentuk yang diharapkan** 

`Dockerfile` untuk backend dan/atau frontend. _Multi-stage build_ sangat dihargai. **Docker** `docker-compose.yml` untuk menjalankan seluruh stack sekaligus adalah nilai plus besar. 

**Item** 

#### **Bentuk yang diharapkan** 

**Kubernetes**<sup>Manifest sederhana:</sup><sup>`Deployment`,</sup><sup>`Service`,</sup><sup>`ConfigMap`/</sup><sup>`Secret`. Tidak perlu Helm</sup> chart, tidak perlu cluster yang benar-benar berjalan. Konfigurasi pipeline (GitHub Actions, GitLab CI, dsb.) yang menjalankan lint → **CI/CD** test → build image. Cukup pipeline yang masuk akal, tidak harus hijau sempurna. 

**Penting:** Jika waktu tidak mencukupi untuk implementasi, **tuliskan pemahaman konseptual Anda di dokumen** — bagaimana aplikasi ini akan Anda kemas, deploy, dan rilis; apa yang perlu diperhatikan soal _environment variable_ , _health check_ , _readiness probe_ , _rollback_ , dan _zerodowntime deployment_ . Penjelasan konseptual yang solid tetap bernilai tinggi. 



## **6. Rubrik Penilaian** 

|**Kriteria**|**Indikator**|
|---|---|
|**Business process &**|Pemetaan alur bisnis, kualitas asumsi, kedalaman analisis edge|
|**problem framing**|case, kemampuan menentukan prioritas scope.|
|**Komunikasi**|Kejelasan dokumen, struktur argumentasi, kualitas README, dan<br>efektivitas penjelasan untuk audiens non-teknis.|
|**Kemampuan teknis inti**|Struktur project, desain API & data model, penanganan error,<br>kualitas kode Golang & React/Vue, kerapian commit.|
|**Infrastructure & delivery**|<sup>Docker, Kubernetes, CI/CD — implementasi maupun pemahaman</sup><br>konseptual._(optional & big bonus)_|
|**Mengerjakan kedua case**<br>**study**|Hanya dihitung apabila kualitas keduanya tetap terjaga._(optional &_<br>_big bonus)_|





## **8. FAQ** 

**Q: Saya tidak sempat menyelesaikan semua scope minimum. Apakah tetap boleh submit?** A: Sangat boleh, dan mohon tetap submit. Tuliskan apa yang belum selesai dan mengapa. Kandidat yang menyelesaikan 60% scope dengan dokumentasi jernih biasanya menempati peringkat lebih tinggi daripada yang menyelesaikan 100% tanpa penjelasan. 

#### **Q: Boleh memakai boilerplate/starter template?** A: Boleh. Sebutkan saja di README. 

**Q: Databasenya harus apa?** A: Bebas. Anda bahkan boleh menggunakan in-memory store, selama konsekuensinya Anda jelaskan. 

**Q: Perlu authentication/login?** A: Tidak perlu diimplementasikan. Cukup jelaskan di dokumen bagaimana Anda akan menanganinya, terutama terkait siapa yang berhak melakukan aksi tertentu. 

**Q: Saya lebih kuat di backend daripada frontend (atau sebaliknya). Bagaimana?** A: Tidak masalah — sebutkan saja secara jujur di README. Kami mengevaluasi kandidat sebagai satu kesatuan, bukan sebagai checklist. 

**Q: Kode saya berantakan, tapi jalan. Apakah lebih baik saya rapikan atau tambah fitur?** A: Rapikan, lalu tulis dokumen 

