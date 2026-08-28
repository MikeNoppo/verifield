-- Dijalankan sekali saat volume Postgres pertama kali dibuat.
--
-- Atlas menghitung diff migrasi dengan cara memutar ulang seluruh berkas di
-- migrations/ pada database kosong, lalu membandingkannya dengan schema yang
-- diinginkan. Database itu harus terpisah dari database aplikasi karena Atlas
-- membuat dan menghapus objek di dalamnya.
--
-- Hanya dipakai saat `atlas migrate diff`; production tidak memerlukannya.
CREATE DATABASE verifield_dev;
