import type { Route } from "next"

/** searchParams Next bisa berisi nilai berulang (?f=a&f=b). Layar ini selalu
    memakai yang pertama; sisanya tidak punya arti di sini. */
export function first(v: string | string[] | undefined): string {
  return Array.isArray(v) ? (v[0] ?? "") : (v ?? "")
}

/** Aktor demo hidup di query string, dan setiap href yang tidak menuliskannya
    ulang akan membuangnya. Tanpa penyisipan ini satu klik ke detail order
    memulangkan identitas ke aktor baku peran, sehingga order milik inspektor
    lain dijawab 404 oleh backend.

    actorId harus aktor yang sudah diselesaikan, bukan nilai mentah dari URL:
    id yang tidak cocok dengan peran halaman akan ditolak saat dibaca ulang,
    dan menyalinnya ke setiap tautan membuat URL dan identitas yang tampil
    berbeda selamanya. */
export function withActor(href: string, actorId: string | null): Route {
  // Tujuan di luar aplikasi ini tidak punya aktor, dan merakit ulang URL-nya
  // dari pathname akan menghapus originnya.
  if (!actorId || /^([a-z][a-z0-9+.-]*:|\/\/)/i.test(href)) return href as Route

  const url = new URL(href, "http://l")
  url.searchParams.set("actor", actorId)
  return `${url.pathname}${url.search}${url.hash}` as Route
}
