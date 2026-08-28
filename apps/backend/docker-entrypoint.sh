#!/bin/sh
# Titik masuk container backend: terapkan migrasi, isi data contoh, lalu jalankan API.
#
# Migrasi dan seeding dibawa masuk ke sini (bukan container Job terpisah) supaya
# `docker compose up` polos cukup menaikkan tiga container: postgres, backend,
# frontend. Kedua langkah idempoten sehingga aman dijalankan ulang setiap restart.
# Pola ini untuk pengembangan/demo dengan satu replika; untuk banyak replika,
# migrasi harus dipindah ke Job satu-kali (lihat deploy/k8s/10-migrate-job.yaml).
set -e

migrate up
seeder

exec api
