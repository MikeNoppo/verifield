"use client"

import { orderFromDTO } from "@/lib/api/orders"
import { API_BASE_URL } from "@/lib/api/client"
import type { JobOrderDTO } from "@verifield/contract"
import type { JobOrder } from "@/lib/domain/types"

export type ConnectionState = "connecting" | "online" | "reconnecting" | "offline"

export type LiveState = {
  /** Order yang sudah diperbarui lewat stream, ditumpangkan di atas hasil render
      server. Hanya berisi yang benar-benar berubah sejak halaman dimuat. */
  orders: ReadonlyMap<string, JobOrder>
  connection: ConnectionState
  /** Waktu pesan terakhir diterima. Ditampilkan agar pengguna tahu seberapa
      mutakhir yang sedang dilihatnya — layar yang diam punya dua arti yang tidak
      bisa dibedakan tanpa ini (F-01). */
  lastMessageAt: string | null
  /** Seq tertinggi yang pernah diterima, dipakai sebagai kursor pemulihan. */
  cursor: number
}

const empty: LiveState = {
  orders: new Map(),
  connection: "offline",
  lastMessageAt: null,
  cursor: 0,
}

let state: LiveState = empty
const listeners = new Set<() => void>()

function publish(next: LiveState) {
  state = next
  for (const listener of listeners) listener()
}

function patch(partial: Partial<LiveState>) {
  publish({ ...state, ...partial })
}

/** Menerapkan satu order ke store.
 *
 *  Pesan ber-seq yang tidak lebih besar dari yang sudah diterapkan DIABAIKAN.
 *  Inilah yang membuat jaminan urutan dan idempotency berlaku di dua sisi: server
 *  sudah menolak event ganda dan event yang datang terbalik, tetapi klien juga
 *  bisa menerima pesan yang sama dua kali — replay saat menyambung ulang sengaja
 *  tumpang tindih dengan siaran langsung agar tidak ada celah. */
export function apply(order: JobOrder): boolean {
  const known = state.orders.get(order.id)
  if (known && known.seq >= order.seq) return false

  const orders = new Map(state.orders)
  orders.set(order.id, order)

  publish({
    ...state,
    orders,
    cursor: Math.max(state.cursor, order.seq),
    lastMessageAt: new Date().toISOString(),
  })
  return true
}

export function subscribe(onChange: () => void): () => void {
  listeners.add(onChange)
  return () => listeners.delete(onChange)
}

export function getSnapshot(): LiveState {
  return state
}

/** Server dan render pertama di klien sama-sama memakai keadaan kosong, sehingga
    markup keduanya identik. Nilai sebenarnya masuk setelah berlangganan. */
export function getServerSnapshot(): LiveState {
  return empty
}

// ---------------------------------------------------------------------------
// Koneksi
// ---------------------------------------------------------------------------

let source: EventSource | null = null
let connections = 0

/** Membuka satu stream untuk seluruh layar, berapa pun jumlah komponen yang
    membacanya. Dihitung dengan penghitung acuan karena HTTP/1.1 membatasi enam
    koneksi per origin — satu stream per komponen akan menghabiskannya, dan
    permintaan biasa ikut tertahan di belakangnya. */
export function connect(actorId: string, onUnknownOrder: () => void): () => void {
  connections += 1

  if (!source) {
    open(actorId, onUnknownOrder)
  }

  return () => {
    connections -= 1
    if (connections > 0) return
    source?.close()
    source = null
    patch({ connection: "offline" })
  }
}

function open(actorId: string, onUnknownOrder: () => void) {
  const params = new URLSearchParams({ actor_id: actorId })
  // EventSource tidak bisa memasang header, jadi identitas dikirim lewat query.
  // Pada koneksi PERTAMA browser juga tidak mengirim Last-Event-ID, sehingga
  // kursor pemulihan harus disertakan sendiri — tanpa ini, perubahan yang
  // terjadi selama halaman dimuat ulang tidak akan pernah tersusul.
  if (state.cursor > 0) params.set("last_event_id", String(state.cursor))

  patch({ connection: "connecting" })

  const es = new EventSource(`${API_BASE_URL}/stream?${params}`)
  source = es

  es.addEventListener("open", () => patch({ connection: "online" }))

  es.addEventListener("order.updated", (event) => {
    const raw = (event as MessageEvent<string>).data
    let order: JobOrder
    try {
      order = orderFromDTO(JSON.parse(raw) as JobOrderDTO)
    } catch {
      return
    }

    const known = state.orders.has(order.id)
    const changed = apply(order)

    // Order yang belum pernah terlihat tidak bisa sekadar ditumpangkan: daftar
    // yang sedang tampil disaring di server, dan menyisipkannya di klien akan
    // melanggar saringan itu. Halaman dimuat ulang agar saringannya tetap benar.
    if (changed && !known) onUnknownOrder()
  })

  es.addEventListener("error", () => {
    // EventSource menyambung ulang sendiri; readyState membedakan "sedang
    // mencoba lagi" dari "menyerah".
    patch({ connection: es.readyState === EventSource.CLOSED ? "offline" : "reconnecting" })
  })
}

/** Menandai koneksi terputus tanpa menutup stream — dipakai saat browser
    melaporkan jaringan hilang, yang lebih cepat diketahui daripada menunggu
    EventSource kehabisan waktu. */
export function markOffline() {
  patch({ connection: "offline" })
}

/** Hanya untuk test. */
export function reset() {
  state = empty
  listeners.clear()
  source?.close()
  source = null
  connections = 0
}
