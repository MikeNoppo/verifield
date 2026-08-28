// Package migrations menyematkan seluruh berkas migrasi ke dalam binary lewat
// go:embed. Karena itu deploy tidak perlu menyalin folder migrations/ maupun
// memasang tool eksternal apa pun — cukup satu binary `migrate`.
package migrations

import "embed"

// FS berisi seluruh berkas .sql di direktori ini.
//
//go:embed *.sql
var FS embed.FS
