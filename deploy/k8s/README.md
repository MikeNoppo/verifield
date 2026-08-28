# Manifest Kubernetes

Manifest minimum untuk menjalankan Verifield di satu namespace. Bukan Helm chart,
dan sengaja tidak memakai templating — tujuannya memperlihatkan keputusan
deployment, bukan menyediakan chart siap pakai.

```bash
kubectl apply -f deploy/k8s/
```

Database diasumsikan berada di luar cluster (RDS, Cloud SQL, atau serupa).
Menjalankan Postgres sebagai Deployment di dalam cluster tidak dilakukan di sini
karena database bersifat stateful dan menuntut StatefulSet beserta
PersistentVolumeClaim — pembahasan tersendiri yang tidak menambah pemahaman
tentang aplikasinya.

## Yang perlu diperhatikan

**Backend menahan koneksi berjam-jam.** Stream SSE tetap terbuka selama klien
membukanya. Karena itu `terminationGracePeriodSeconds` diberi 45 detik: saat
rolling update, pod lama perlu waktu menutup koneksinya dengan rapi. Klien
kemudian menyambung ulang sendiri dan memulihkan perubahan yang terlewat lewat
kursor `seq` — tidak ada data yang hilang, hanya jeda beberapa detik.

**Fan-out tidak bergantung pada sticky session.** Setiap pod mendengarkan kanal
`LISTEN/NOTIFY` Postgres yang sama, sehingga klien boleh mendarat di pod mana
pun. Ini yang membuat `replicas: 3` benar-benar bekerja tanpa session affinity
di Ingress.

**Readiness memisahkan "hidup" dari "siap".** `livenessProbe` hanya memastikan
proses tidak menggantung; `readinessProbe` menembak `/health`, yang baru menjawab
setelah koneksi database berhasil. Pod yang databasenya belum terjangkau tidak
akan menerima lalu lintas.

**Migrasi berjalan sebagai Job, bukan initContainer.** initContainer akan
menjalankan migrasi sekali per pod, sehingga tiga replika berarti tiga eksekusi
bersamaan yang saling berebut. Job memastikan migrasi berjalan tepat satu kali
per rilis.

## Yang belum ada di sini

- **Ingress dan TLS** — bergantung pada ingress controller yang dipakai.
- **HorizontalPodAutoscaler** — jumlah koneksi SSE yang terbuka adalah metrik
  yang lebih tepat daripada CPU untuk beban seperti ini, dan itu menuntut
  metrics adapter tersendiri.
- **NetworkPolicy** — pada sistem sungguhan, hanya backend yang boleh menjangkau
  database.
- **Secret sungguhan** — nilai di `secret.yaml` adalah contoh. Di produksi ia
  datang dari External Secrets, Sealed Secrets, atau penyedia cloud, dan tidak
  pernah ikut masuk ke repositori.
