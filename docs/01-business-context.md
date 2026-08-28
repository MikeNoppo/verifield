# Verifield — Dokumen Konteks Bisnis

**Case Study 1: Real-Time Order & Job Tracking**
Dokumen ini adalah lapisan bisnis. Belum membahas teknologi, arsitektur, atau skema database.
Dokumen ini menjadi sumber kebenaran untuk seluruh keputusan teknis yang menyusul.

---

## 1. Ringkasan Eksekutif

**PT Sentra Inspeksi Nusantara** adalah perusahaan jasa inspeksi dan sampling lapangan. Klien memesan pemeriksaan atas suatu objek — kargo di pelabuhan, tumpukan batu bara di tambang, tangki penyimpanan di kilang — dan perusahaan mengirimkan inspektor bersertifikat ke lokasi untuk melakukan pemeriksaan dan pengambilan sampel.

Saat ini tim Customer Service kewalahan menjawab satu pertanyaan yang sama berulang kali: *"Pekerjaan saya sudah sampai mana?"*

**Verifield** adalah sistem yang membuat status pekerjaan terlihat langsung oleh klien tanpa perantara manusia, dan sekaligus memberi inspektor lapangan cara memperbarui status yang tetap bekerja di area dengan sinyal buruk.

---

## 2. Mengapa Domain Ini Yang Dipilih

Deskripsi case sengaja dibuat generik ("platform layanan operasional / jasa lapangan"). Saya memilih mengkonkretkannya menjadi jasa inspeksi lapangan karena tiga alasan berikut.

**Pertama, tantangan konektivitas di soal menjadi masalah nyata, bukan skenario yang dikarang.**
Soal menyebut bahwa teknisi bekerja *"seringkali dari perangkat mobile dengan koneksi tidak stabil"*. Pada domain seperti servis peralatan rumah tangga, asumsi ini terasa dipaksakan — sinyal di area perumahan umumnya memadai. Pada inspeksi lapangan, lokasi kerja berada di area tambang, dermaga, gudang berdinding logam, atau perkebunan, di mana kehilangan sinyal selama berjam-jam adalah kondisi harian. Dengan demikian, kebutuhan akan penanganan *missed events*, *idempotency*, dan *out-of-order update* lahir dari kondisi bisnis, bukan dari keinginan merapikan teknis.

**Kedua, pembatalan memiliki biaya yang nyata dan terukur.**
Inspektor dapat menempuh perjalanan empat jam menuju lokasi tambang. Ketika klien membatalkan di tengah perjalanan, kerugiannya konkret: waktu inspektor, biaya transportasi, dan slot jadwal yang tidak bisa dijual ke klien lain. Ini memberi dasar rasional bagi aturan "siapa boleh membatalkan sampai titik mana", sehingga aturan tersebut menjadi keputusan bisnis yang dapat dipertanggungjawabkan, bukan pembatasan sewenang-wenang.

**Ketiga, urutan kejadian memiliki konsekuensi hukum dan komersial.**
Hasil inspeksi menjadi dasar penerbitan sertifikat, klaim asuransi, dan pelepasan pembayaran dalam transaksi perdagangan. Waktu kejadian bukan sekadar informasi; ia dapat dipersengketakan. Ini memberi bobot nyata pada kebutuhan *audit trail* dan pemisahan antara waktu kejadian di lapangan dan waktu penerimaan di server.

**Catatan sikap.** Domain ini diangkat sebagai perusahaan jasa inspeksi pada umumnya, berdasarkan karakteristik yang disebutkan dalam soal. Dokumen ini tidak mengklaim menggambarkan proses bisnis internal perusahaan mana pun secara spesifik.

---

## 3. Profil Bisnis

| Aspek | Keterangan |
|---|---|
| Nama perusahaan | PT Sentra Inspeksi Nusantara (fiktif) |
| Nama sistem | Verifield |
| Bidang | Jasa inspeksi, sampling, dan verifikasi lapangan |
| Model pendapatan | Biaya per penugasan (per job), dihitung dari jenis inspeksi, lokasi, dan durasi |
| Pelanggan | Eksportir, importir, trader komoditas, perusahaan tambang, perusahaan logistik |
| Objek yang diperiksa | Kargo curah, kontainer, tangki, stockpile, hasil pertanian |
| Skala operasi (asumsi) | 40–60 job order per hari, 25 inspektor aktif, 3 kota operasi |
| Karakteristik pekerjaan | Berlokasi di luar kantor, bergantung jadwal pihak ketiga, sering di area sinyal lemah |

**Rantai nilainya:** klien memesan → koordinator menugaskan inspektor → inspektor berangkat, memeriksa, mengambil sampel → hasil dilaporkan → sertifikat diterbitkan.

Verifield menangani bagian tengah rantai ini, yaitu dari pemesanan sampai pekerjaan lapangan dinyatakan selesai. Penerbitan sertifikat dan pengujian laboratorium berada di luar cakupan.

---

## 4. Masalah Saat Ini dan Akar Masalahnya

### Gejala yang terlihat

Tim Customer Service menghabiskan sebagian besar waktunya menjawab pertanyaan status. Beban ini bertambah pada hari sibuk, dan klien besar sering menelepon beberapa kali dalam satu hari untuk satu order yang sama.

### Akar masalah yang sebenarnya

Rumusan dangkal atas masalah ini adalah *"klien tidak bisa melihat status pekerjaannya"*. Rumusan tersebut kurang tepat, dan solusinya akan salah sasaran.

Akar masalah yang sesungguhnya adalah: **status pekerjaan hidup di kepala inspektor, bukan di dalam sistem.**

Customer Service bukan sekadar penjawab telepon. CS berfungsi sebagai *jembatan manual* antara lapangan dan klien. Ketika klien bertanya, CS sering harus lebih dulu menghubungi inspektor melalui telepon atau pesan singkat, menunggu balasan, baru kemudian menjawab klien. Satu pertanyaan klien menghasilkan dua sampai tiga interaksi manual.

### Konsekuensi penting dari rumusan ini

Membangun halaman tracking untuk klien **tidak akan menyelesaikan masalah** apabila inspektor tidak memperbarui status secara disiplin. Apabila data yang tampil di layar sudah basi, klien tetap akan menelepon CS — bahkan berpotensi lebih sering, karena kini mereka memiliki bukti konkret bahwa status tidak berubah selama tiga jam.

Maka masalah ini memiliki dua sisi yang harus ditangani bersamaan:

1. **Sisi klien** — membutuhkan visibilitas yang dapat dipercaya tanpa perantara.
2. **Sisi inspektor** — membutuhkan cara memperbarui status yang biayanya sangat murah: satu ketukan, tetap bekerja saat sinyal hilang, dan tidak menghukum pengguna apabila tombol tertekan berulang kali.

Sisi kedua inilah yang menentukan keberhasilan sistem, dan sisi inilah yang biasanya diabaikan.

**Ukuran keberhasilan yang diusulkan:** penurunan jumlah panggilan bertanya status, dan proporsi job order yang statusnya diperbarui inspektor dalam waktu kurang dari 15 menit sejak kejadian sebenarnya.

---

## 5. Aktor dan Tanggung Jawabnya

### 5.1 Klien

Perwakilan perusahaan pemesan jasa. Bukan orang teknis. Umumnya membuka sistem dari komputer kantor, sesekali dari ponsel.

**Yang dikerjakan:**
- Membuat permintaan inspeksi baru
- Memantau status seluruh order miliknya
- Melihat riwayat perubahan status satu order
- Mengajukan pembatalan

**Yang menjadi kebutuhan utamanya:** mengetahui apakah inspektor sudah berangkat dan kapan perkiraan selesai, tanpa perlu menelepon siapa pun.

**Batasan:** hanya dapat melihat order milik perusahaannya sendiri. Tidak dapat melihat identitas lengkap atau posisi inspektor.

### 5.2 Koordinator Operasional (Admin / Dispatcher)

Pengguna internal. Bekerja dari kantor dengan koneksi stabil. Memantau seluruh order berjalan pada satu layar sepanjang hari.

**Yang dikerjakan:**
- Menugaskan inspektor pada job order
- Memantau seluruh order aktif
- Menyetujui atau menolak permintaan pembatalan yang diajukan setelah pekerjaan dimulai
- Melakukan koreksi status apabila terjadi kesalahan input di lapangan
- Menindaklanjuti order yang tidak menunjukkan pembaruan dalam waktu lama

**Yang menjadi kebutuhan utamanya:** mengetahui secepat mungkin order mana yang bermasalah, tanpa harus memeriksa satu per satu.

### 5.3 Inspektor Lapangan

Pengguna dengan kondisi kerja paling sulit. Bekerja sambil berdiri, kadang mengenakan sarung tangan atau alat pelindung diri, sering di bawah matahari langsung, dan dengan sinyal yang tidak dapat diandalkan.

**Yang dikerjakan:**
- Melihat daftar penugasan hari ini
- Memperbarui status: berangkat, tiba di lokasi, mulai bekerja, selesai
- Melaporkan kegagalan pelaksanaan beserta alasannya

**Yang menjadi kebutuhan utamanya:** memperbarui status dengan satu ketukan, dan mendapat kepastian bahwa laporannya tidak hilang meskipun sinyal sedang tidak ada.

**Yang tidak boleh dilakukan:** membatalkan order. Alasannya dijelaskan pada keputusan B-04.

### 5.4 Customer Service

Bukan pengguna utama sistem ini, melainkan pihak yang bebannya hendak dikurangi. CS memiliki akses baca yang sama dengan koordinator, digunakan pada saat klien tetap menghubungi melalui telepon.

---

## 6. Objek Bisnis Utama

### Job Order

Satu permintaan pemeriksaan, atas satu objek, di satu lokasi, pada satu rentang waktu.

Informasi yang melekat: nomor referensi, klien pemesan, jenis inspeksi, lokasi, jadwal yang diminta, inspektor yang ditugaskan, status saat ini, dan riwayat lengkap perubahan status.

### Riwayat Status (Status History)

Catatan setiap kejadian perubahan status: dari status apa ke status apa, oleh siapa, kapan kejadiannya di lapangan, kapan diterima sistem, dan apa alasannya bila ada.

**Riwayat ini bersifat menambah, tidak pernah menimpa atau menghapus.** Ini adalah keputusan paling mendasar dalam dokumen ini, dan alasannya dijelaskan pada keputusan B-01.

---

## 7. Siklus Status

Soal memberi contoh `To Do → In Progress → Done`. Siklus tersebut saya perluas menjadi berikut.

```
   Requested
       │  (koordinator menugaskan inspektor)
       ▼
    Assigned ─────────────────────────┐
       │  (inspektor berangkat)       │
       ▼                              │
   On The Way ────────────────────────┤
       │  (inspektor tiba di lokasi)  │
       ▼                              │
    On Site ───────────────┬──────────┤
       │  (mulai bekerja)  │          │
       ▼                   │          │
  In Progress ─────────────┤          │
       │  (pekerjaan usai) │          │
       ▼                   ▼          ▼
   Completed            Failed    Cancelled
```

`Completed`, `Failed`, dan `Cancelled` adalah status final. Tidak ada transisi keluar dari ketiganya.

### Arti setiap status

| Status | Arti bagi bisnis | Yang mengubah |
|---|---|---|
| `Requested` | Order diterima, inspektor belum ditentukan | Sistem, saat klien memesan |
| `Assigned` | Inspektor sudah ditetapkan, belum berangkat | Koordinator |
| `On The Way` | Inspektor dalam perjalanan menuju lokasi | Inspektor |
| `On Site` | Inspektor tiba di lokasi, pekerjaan belum dimulai | Inspektor |
| `In Progress` | Pemeriksaan atau sampling sedang berlangsung | Inspektor |
| `Completed` | Pekerjaan lapangan selesai | Inspektor |
| `Failed` | Inspektor tiba, pekerjaan tidak dapat dilaksanakan | Inspektor |
| `Cancelled` | Order dibatalkan sebelum atau saat pelaksanaan | Klien atau koordinator |

### Mengapa lebih rinci daripada contoh di soal

**`On The Way` dan `On Site` dipisahkan.** Pertanyaan yang paling sering diajukan klien bukan *"sudah selesai belum"*, melainkan *"orangnya sudah berangkat belum"* dan *"sudah sampai lokasi belum"*. Siklus status yang tidak mampu menjawab pertanyaan tersering tidak akan mengurangi beban Customer Service, sehingga tujuan bisnis sistem ini tidak tercapai. Kedua status ini ada karena tujuan bisnisnya, bukan karena kelengkapan diagram.

**`Failed` dipisahkan dari `Cancelled`.** Keduanya sama-sama berarti pekerjaan tidak selesai, tetapi berbeda secara komersial dan berbeda pihak yang bertanggung jawab.

- `Cancelled` adalah keputusan pihak pemesan atau perusahaan.
- `Failed` adalah kondisi lapangan yang menghalangi pelaksanaan: kargo belum tiba di dermaga, akses lokasi ditolak, cuaca membahayakan, atau objek yang akan diperiksa tidak ditemukan.

Tanpa status `Failed`, inspektor akan terpaksa memakai `Cancelled` untuk situasi yang bukan pembatalan. Akibatnya data laporan menjadi kotor, dan perusahaan kehilangan kemampuan mengukur berapa banyak kunjungan yang gagal karena kesiapan klien — padahal angka itu adalah dasar untuk menagih biaya kunjungan sia-sia.

---

## 8. Alur Kerja Bisnis Utama

### Alur normal, dari awal sampai selesai

```
KLIEN                 KOORDINATOR              INSPEKTOR              SISTEM
  │                        │                       │                     │
  ├─ buat permintaan ──────────────────────────────────────────────────► Requested
  │                        │                       │                     │
  │                        ├─ tugaskan inspektor ──────────────────────► Assigned
  │                        │                       │                     │
  │                        │                       ├─ tekan Berangkat ─► On The Way
  │  ◄──── status berubah langsung, tanpa refresh ─────────────────────┤
  │                        │                       │                     │
  │                        │                       ├─ tekan Tiba ──────► On Site
  │                        │                       │                     │
  │                        │                       ├─ tekan Mulai ─────► In Progress
  │                        │                       │                     │
  │                        │                       ├─ tekan Selesai ───► Completed
  │  ◄──── notifikasi selesai ─────────────────────────────────────────┤
```

### Alur pembatalan

```
KLIEN mengajukan pembatalan
  │
  ├── status masih Requested / Assigned
  │      └──► langsung Cancelled, tanpa biaya
  │
  ├── status On The Way / On Site
  │      └──► langsung Cancelled, dikenakan biaya kunjungan
  │
  └── status In Progress
         └──► BUKAN pembatalan langsung.
              Menjadi permintaan pembatalan yang menunggu keputusan koordinator.
              │
              ├─ koordinator setuju ──► Cancelled
              └─ koordinator tolak ───► kembali berjalan sebagai In Progress
```

### Alur inspektor kehilangan sinyal

```
Inspektor menekan "Tiba di Lokasi" pada pukul 09.14, sinyal tidak ada
  │
  ├─ aplikasi menyimpan kejadian secara lokal, menandai "menunggu terkirim"
  ├─ inspektor melanjutkan bekerja, menekan "Mulai" 09.20 dan "Selesai" 11.05
  │
  └─ pukul 11.40 sinyal kembali
        │
        └─ tiga kejadian terkirim sekaligus, membawa waktu kejadian masing-masing
              │
              ├─ sistem mengurutkan berdasarkan waktu kejadian, bukan waktu tiba
              ├─ sistem mengabaikan kiriman ganda berdasarkan penanda unik
              └─ riwayat klien menampilkan urutan yang benar: 09.14, 09.20, 11.05
```

---

## 9. Alur per Fitur

### F-01 — Klien memantau daftar order

Klien membuka halaman daftar. Sistem menampilkan seluruh order milik perusahaannya beserta status terkini. Ketika inspektor memperbarui status di lapangan, baris pada daftar berubah tanpa klien melakukan apa pun.

Halaman menampilkan indikator koneksi sederhana dengan tiga kondisi: tersambung, sedang menyambung ulang, dan terputus. Ketika terputus, sistem menampilkan waktu pembaruan terakhir agar klien tahu bahwa yang dilihatnya mungkin sudah tidak mutakhir.

**Alasan indikator koneksi diperlukan:** tanpa indikator ini, layar yang diam memiliki dua arti yang tidak dapat dibedakan — pekerjaan memang belum berubah, atau sambungan terputus. Perbedaan tersebut menentukan apakah klien perlu menelepon atau tidak. Menghilangkan ambiguitas ini secara langsung mendukung tujuan mengurangi beban CS.

### F-02 — Klien melihat detail dan riwayat

Klien membuka satu order dan melihat garis waktu perubahan status secara berurutan, lengkap dengan waktu setiap tahap. Garis waktu ini juga berfungsi sebagai bukti apabila kemudian terjadi perselisihan mengenai kapan pekerjaan dilaksanakan.

Waktu yang ditampilkan adalah waktu kejadian di lapangan, bukan waktu penerimaan sistem. Alasannya dijelaskan pada keputusan B-02.

### F-03 — Koordinator menugaskan inspektor

Koordinator melihat order berstatus `Requested`, memilih inspektor, dan menugaskannya. Status berubah menjadi `Assigned`, dan perubahan tersebut langsung terlihat oleh klien serta muncul pada daftar penugasan inspektor.

Pada PoC, pemilihan inspektor dilakukan manual dari daftar. Penugasan otomatis berdasarkan ketersediaan dan jarak berada di luar cakupan.

### F-04 — Inspektor memperbarui status

Layar inspektor menampilkan satu tombol besar berisi tindakan berikutnya yang mungkin dilakukan. Inspektor tidak memilih status dari daftar, melainkan menekan satu tombol yang sudah sesuai konteks.

**Alasan rancangan ini:** inspektor bekerja sambil berdiri, sering mengenakan sarung tangan, dan tidak akan membaca petunjuk. Daftar pilihan status membuka peluang salah pilih, dan setiap kesalahan input harus dikoreksi koordinator secara manual di kemudian hari. Satu tombol yang sudah benar secara konteks menghilangkan seluruh kelas kesalahan tersebut.

Ketika tombol ditekan sementara sinyal tidak tersedia, aplikasi tetap menerima dan menyimpan kejadian secara lokal, memberi tanda "menunggu terkirim", dan mengirimkannya ketika sambungan kembali.

### F-05 — Pembatalan

Klien menekan tombol batal. Sistem memeriksa status saat ini dan menentukan hasilnya sesuai matriks pada bagian 10. Apabila pembatalan tidak diizinkan pada tahap tersebut, sistem menolak dengan penjelasan yang dapat dimengerti orang non-teknis, bukan sekadar pesan kesalahan.

Contoh penolakan yang baik: *"Pekerjaan sudah dimulai di lokasi. Permintaan pembatalan Anda telah kami teruskan ke koordinator untuk ditinjau."*

### F-06 — Koreksi oleh koordinator

Koordinator dapat mengoreksi status yang salah, termasuk mengembalikannya ke tahap sebelumnya. Koreksi wajib disertai alasan, dan tercatat sebagai entri baru pada riwayat, bukan menghapus entri lama.

**Alasan:** kesalahan input di lapangan pasti terjadi. Tanpa mekanisme koreksi resmi, koordinator akan mengubah data langsung ke basis data, dan seluruh nilai audit trail hilang.

---

## 10. Keputusan Bisnis dan Sistem

Setiap keputusan diberi nomor agar dapat dirujuk dari dokumen teknis, tabel edge case, dan README repositori.

---

**B-01 — Status disimpan sebagai riwayat kejadian, bukan sebagai satu nilai yang ditimpa**

Status terkini adalah hasil turunan dari riwayat, bukan sumber kebenaran itu sendiri.

*Alasan:* pekerjaan ini menghasilkan dokumen yang dipakai dalam klaim dan transaksi komersial. Pertanyaan "kapan inspektor tiba di lokasi" dapat dipersengketakan. Apabila status hanya berupa satu kolom yang ditimpa, jawabannya hilang setiap kali status berubah. Selain itu, riwayat kejadian adalah satu-satunya cara memenuhi tiga kebutuhan lain sekaligus: pengiriman ulang kejadian yang terlewat, deteksi kejadian yang datang terbalik urutannya, dan audit.

*Konsekuensi:* volume data lebih besar, dan pembacaan status terkini memerlukan penanganan khusus agar tetap cepat.

---

**B-02 — Waktu kejadian dan waktu penerimaan dicatat terpisah**

Setiap perubahan status membawa dua penanda waktu: kapan kejadiannya terjadi di lapangan, dan kapan sistem menerimanya.

*Alasan:* di area tanpa sinyal, jeda antara keduanya dapat mencapai beberapa jam. Untuk laporan, penagihan, dan pengukuran ketepatan layanan, yang sah adalah waktu kejadian. Untuk pengurutan teknis dan penelusuran masalah, yang diperlukan adalah waktu penerimaan. Menggabungkan keduanya menjadi satu kolom berarti kehilangan salah satu kebenaran.

*Konsekuensi:* waktu kejadian berasal dari perangkat inspektor, sehingga tunduk pada jam perangkat yang bisa saja tidak akurat. Perlu ada batas kewajaran, misalnya menolak waktu kejadian yang berada di masa depan atau terlalu jauh di masa lalu.

---

**B-03 — Setiap pembaruan dari lapangan membawa penanda unik yang dibuat di perangkat**

*Alasan:* inspektor dengan sinyal lemah akan menekan tombol berulang kali karena layar tidak segera merespons. Ini bukan kesalahan pengguna, melainkan perilaku manusia yang dapat diperkirakan. Tanpa penanda unik, satu kejadian nyata akan tercatat lima kali, dan garis waktu yang dilihat klien menjadi tidak masuk akal.

*Konsekuensi:* perangkat harus mampu menghasilkan penanda unik secara mandiri tanpa meminta ke server, karena pada saat itu server tidak terjangkau.

---

**B-04 — Inspektor tidak berwenang membatalkan order**

Inspektor hanya dapat melaporkan `Failed` disertai alasan.

*Alasan:* apabila inspektor dapat membatalkan sendiri, terbentuk insentif untuk menghindari pekerjaan yang sulit, jauh, atau tidak nyaman. Pemisahan wewenang ini adalah kendali internal yang standar dalam operasi lapangan. Selain itu, `Cancelled` memiliki konsekuensi komersial terhadap klien, dan keputusan komersial bukan wewenang pelaksana lapangan.

---

**B-05 — Pembatalan setelah pekerjaan dimulai menjadi permintaan, bukan tindakan langsung**

*Alasan:* setelah status `In Progress`, sudah ada biaya nyata yang dikeluarkan perusahaan: waktu inspektor, perjalanan, dan kemungkinan sampel yang telah diambil. Membiarkan klien membatalkan secara sepihak memindahkan seluruh kerugian kepada perusahaan tanpa proses. Meninjau permintaan tersebut memungkinkan koordinator menyelesaikan aspek komersialnya sebelum order ditutup.

---

**B-06 — Transisi status hanya boleh maju; koreksi mundur adalah wewenang koordinator**

*Alasan:* pembaruan yang datang terlambat dan keluar urutan tidak boleh membuat status yang dilihat klien mundur. Klien yang melihat statusnya kembali dari `Completed` ke `On Site` akan kehilangan kepercayaan pada sistem, dan justru menelepon CS — kebalikan dari tujuan sistem ini. Koreksi tetap dimungkinkan, tetapi melalui jalur resmi yang tercatat dan beralasan.

---

**B-07 — Pembaruan yang datang setelah status final ditolak, tetapi tetap dicatat dan dilaporkan**

*Alasan:* skenario ini nyata dan penting. Klien membatalkan order ketika inspektor sedang offline. Inspektor tidak mengetahuinya, tetap mengerjakan pemeriksaan, dan laporan `Completed` masuk ketika sinyal kembali. Status tidak boleh berubah, karena `Cancelled` bersifat final.

Namun menolak secara diam-diam adalah kesalahan. Ada pekerjaan nyata yang telah dilakukan seseorang. Kejadian tersebut dicatat pada riwayat sebagai pembaruan terlambat yang ditolak, dan koordinator diberi tanda agar dapat menyelesaikan kompensasi bagi inspektor serta menjelaskan situasinya kepada klien.

*Alasan mendasar:* sistem yang menelan pekerjaan penggunanya tanpa penjelasan akan ditinggalkan penggunanya. Inspektor akan kembali melapor lewat telepon, dan masalah awal muncul lagi.

---

**B-08 — Klien hanya melihat status, tidak melihat posisi atau identitas lengkap inspektor**

*Alasan:* posisi seseorang adalah data pribadi pekerja. Kebutuhan bisnis klien adalah mengetahui kemajuan pekerjaan, dan kebutuhan tersebut sudah terpenuhi oleh status. Menampilkan posisi berarti mengumpulkan dan membagikan data yang tidak diperlukan untuk tujuannya.

---

**B-09 — Ketika dua koordinator mengubah order yang sama secara bersamaan, perubahan pertama yang menang**

Koordinator kedua menerima penolakan disertai penjelasan bahwa data telah berubah, dan tampilannya diperbarui.

*Alasan:* menerima kedua perubahan berarti kehilangan salah satunya secara diam-diam, dan tidak ada yang menyadari data telah hilang. Penolakan yang terlihat jauh lebih baik daripada kehilangan yang tidak terlihat, terutama pada data yang menjadi dasar dokumen komersial.

---

## 11. Matriks Kewenangan Pembatalan

| Status saat ini | Klien | Koordinator | Inspektor | Konsekuensi komersial |
|---|---|---|---|---|
| `Requested` | Boleh | Boleh | Tidak | Tanpa biaya |
| `Assigned` | Boleh | Boleh | Tidak | Tanpa biaya |
| `On The Way` | Boleh | Boleh | Tidak | Biaya perjalanan |
| `On Site` | Boleh | Boleh | Tidak | Biaya kunjungan |
| `In Progress` | Hanya mengajukan | Boleh | Tidak | Ditentukan koordinator |
| `Completed` | Tidak | Tidak | Tidak | Status final |
| `Failed` | Tidak | Tidak | Tidak | Status final |
| `Cancelled` | Tidak | Tidak | Tidak | Status final |

---

## 12. Skenario Tidak Ideal dan Penanganannya

| Skenario | Risiko bagi bisnis | Penanganan sistem | Keputusan terkait |
|---|---|---|---|
| Inspektor kehilangan sinyal tiga jam, lalu empat pembaruan terkirim sekaligus dengan urutan acak | Garis waktu klien menjadi kacau, waktu pelaksanaan tidak dapat dipertanggungjawabkan | Pengurutan berdasarkan waktu kejadian, bukan waktu penerimaan | B-02, B-06 |
| Inspektor menekan tombol "Selesai" lima kali karena layar tidak merespons | Satu kejadian tercatat lima kali, riwayat tidak masuk akal | Penanda unik dari perangkat, kiriman kedua dan seterusnya diabaikan | B-03 |
| Klien membatalkan saat inspektor offline; inspektor tetap menyelesaikan pekerjaan | Pekerjaan nyata tidak tercatat, inspektor dirugikan, klien tidak memahami situasi | Status tidak berubah, kejadian dicatat sebagai pembaruan terlambat yang ditolak, koordinator diberi tanda | B-07 |
| Inspektor tiba, kargo belum sampai di dermaga | Apabila dicatat sebagai pembatalan, perusahaan kehilangan dasar menagih biaya kunjungan | Status `Failed` dengan alasan, terpisah dari `Cancelled` | Bagian 7 |
| Dua koordinator menugaskan inspektor berbeda pada order yang sama | Satu penugasan hilang tanpa disadari, dua inspektor berangkat ke lokasi yang sama | Perubahan pertama menang, yang kedua ditolak dengan penjelasan | B-09 |
| Klien membuka sistem di dua tab sekaligus | Kedua tab menampilkan data berbeda, klien tidak tahu mana yang benar | Kedua tab menerima pembaruan yang sama, dan keduanya menampilkan indikator koneksi | F-01 |
| Order tidak menunjukkan pembaruan selama delapan jam pada hari kerja | Kemungkinan inspektor lupa memperbarui, dan klien akan menelepon | Koordinator diberi tanda agar dapat menindaklanjuti sebelum klien bertanya | Bagian 4 |
| Inspektor salah menekan "Selesai" padahal baru tiba | Data laporan salah, klien menerima informasi keliru | Koordinator mengoreksi dengan alasan, tercatat sebagai entri baru | B-06, F-06 |
| Jam pada perangkat inspektor tidak akurat | Waktu kejadian keliru dan merusak urutan riwayat | Waktu kejadian di luar batas kewajaran ditolak, waktu penerimaan dipakai sebagai cadangan | B-02 |

---

## 13. Cakupan Pekerjaan

### Termasuk dalam PoC

- Pembuatan job order beserta data awal
- Penugasan inspektor oleh koordinator
- Pembaruan status oleh inspektor sepanjang siklus
- Pembatalan dengan validasi transisi dan matriks kewenangan
- Riwayat status lengkap yang bersifat menambah
- Pembaruan langsung ke layar klien tanpa muat ulang
- Indikator status koneksi
- Data contoh agar sistem dapat langsung dicoba

### Sengaja tidak dikerjakan

| Yang tidak dikerjakan | Alasan |
|---|---|
| Autentikasi dan manajemen pengguna | Dinyatakan tidak dinilai dalam soal. Peran akan diwakili melalui pemilih peran sederhana, dan rancangan wewenangnya dijelaskan di dokumen |
| Penugasan otomatis dan optimasi rute | Masalah optimasi yang terpisah, tidak menambah pemahaman atas tantangan real-time |
| Notifikasi push, surel, dan pesan singkat | Saluran keluar tambahan; mekanisme intinya sudah terwakili oleh pembaruan langsung |
| Pembayaran, penagihan, dan invoice | Berada di hilir proses, di luar rantai yang ditangani sistem ini |
| Pelacakan posisi inspektor secara langsung | Keputusan B-08, dan menambah kompleksitas privasi tanpa menjawab kebutuhan utama |
| Unggah foto dan dokumen hasil pemeriksaan | Menambah penyimpanan berkas dan sinkronisasi berkas offline, yang merupakan masalah tersendiri |
| Penanganan multi-bahasa dan multi-zona waktu | Operasi diasumsikan dalam satu zona waktu |

### Dipertimbangkan namun dikeluarkan

**Status `Awaiting Lab Result`.** Dalam praktik nyata, setelah sampel diambil, pekerjaan tertahan menunggu hasil laboratorium yang dapat memakan waktu beberapa hari dan berada di luar kendali inspektor. Status ini dikeluarkan dari cakupan karena memperkenalkan aktor eksternal dan alur asinkron kedua, sementara nilai pembelajarannya tumpang tindih dengan mekanisme real-time yang sudah ada. Apabila diberi waktu lebih panjang, ini adalah status pertama yang akan ditambahkan.

**Multi-inspektor dalam satu order.** Pemeriksaan berskala besar dapat melibatkan beberapa inspektor sekaligus. Ini dikeluarkan karena menimbulkan pertanyaan yang belum terjawab mengenai siapa yang berwenang mengubah status dan bagaimana status gabungan ditentukan — kompleksitas yang tidak sebanding dengan nilainya bagi PoC ini.

---

## 14. Asumsi yang Diambil

| Kode | Asumsi | Alasan |
|---|---|---|
| A-01 | Satu job order ditangani satu inspektor | Menghindari pertanyaan status gabungan dan wewenang ganda |
| A-02 | Satu job order terikat pada satu klien dan satu lokasi | Sesuai praktik penagihan per penugasan |
| A-03 | Klien hanya melihat order milik perusahaannya | Kerahasiaan komersial antar klien |
| A-04 | Operasi berlangsung dalam satu zona waktu | Menyederhanakan penanganan waktu tanpa mengurangi inti masalah |
| A-05 | Perangkat inspektor mampu menyimpan data secara lokal saat offline | Prasyarat agar pembaruan tidak hilang di area tanpa sinyal |
| A-06 | Jumlah order aktif bersamaan berada pada kisaran puluhan, bukan puluhan ribu | Memengaruhi pilihan teknis, dinyatakan agar konsekuensinya jelas |
| A-07 | Klien mengakses dari komputer kantor; inspektor dari ponsel | Menentukan prioritas rancangan antarmuka masing-masing |
| A-08 | Koreksi status oleh koordinator adalah kejadian yang jarang, bukan alur rutin | Membenarkan rancangan yang sederhana untuk fitur ini |

---

## 15. Catatan untuk Tahap Berikutnya

Dokumen ini menetapkan lapisan bisnis. Keputusan teknis yang menyusul harus dapat dirujuk kembali ke salah satu keputusan di atas.

Beberapa keterkaitan yang perlu dijaga:

- Pemilihan mekanisme real-time harus mampu mendukung pengiriman kejadian yang terlewat setelah sambungan pulih, sebagaimana dituntut oleh B-01 dan F-01
- Rancangan penyimpanan harus memisahkan riwayat kejadian dari status terkini, sesuai B-01
- Setiap kejadian harus menyimpan dua penanda waktu dan satu penanda unik dari perangkat, sesuai B-02 dan B-03
- Validasi transisi status harus dilakukan di sisi server, bukan hanya di antarmuka, karena pembaruan dapat datang terlambat dari perangkat yang belum mengetahui status terkini