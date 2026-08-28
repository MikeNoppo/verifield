"use client"

import { submitStatusEvent } from "@/lib/api/orders"
import { ApiError } from "@/lib/api/client"
import { apply } from "@/lib/live/store"
import type { Status } from "@/lib/domain/types"

/** Satu laporan yang sudah ditekan inspektor tetapi belum tentu terkirim.
 *
 *  Isinya lengkap dan berdiri sendiri: begitu tombol ditekan, laporan ini tidak
 *  lagi bergantung pada apa pun yang ada di layar. Itulah syarat agar ia selamat
 *  melewati sinyal yang hilang, aplikasi yang ditutup, dan halaman yang dimuat
 *  ulang. */
export type OutboxItem = {
  /** Dibuat di perangkat, sebelum kiriman pertama. Server memakainya untuk
      mengenali kiriman berulang, sehingga tombol yang ditekan lima kali karena
      layar tidak merespons tetap menghasilkan satu baris riwayat (B-03). */
  clientEventId: string
  orderId: string
  orderRef: string
  toStatus: Status
  /** Waktu tombol ditekan di lapangan — bukan waktu laporan berhasil terkirim.
      Inilah yang sah untuk laporan dan penagihan (B-02). */
  occurredAt: string
  reason?: string
  actorId: string
  attempts: number
}

export type OutboxState = {
  items: OutboxItem[]
  /** Laporan yang ditolak server secara permanen. Disimpan supaya inspektor tahu
      apa yang terjadi — laporan yang hilang tanpa penjelasan akan membuatnya
      kembali melapor lewat telepon, yang justru masalah awal sistem ini. */
  failures: Array<{ orderRef: string; message: string }>
  flushing: boolean
}

const KEY = "verifield.outbox.v1"

let state: OutboxState = { items: [], failures: [], flushing: false }
const listeners = new Set<() => void>()
let loaded = false

function publish(next: OutboxState) {
  state = next
  for (const listener of listeners) listener()
}

function persist(items: OutboxItem[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(items))
  } catch {
    // Penyimpanan penuh atau diblokir. Antrean tetap hidup di memori, jadi
    // laporan yang sedang menunggu tidak hilang selama halaman belum ditutup.
  }
}

function load(): OutboxItem[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as OutboxItem[]) : []
  } catch {
    return []
  }
}

/** Memulihkan antrean dari penyimpanan perangkat. Dipanggil sekali saat layar
    inspektor dibuka — termasuk setelah aplikasi ditutup dan dibuka lagi. */
export function hydrate() {
  if (loaded) return
  loaded = true
  publish({ ...state, items: load() })
}

export function subscribe(onChange: () => void): () => void {
  listeners.add(onChange)
  return () => listeners.delete(onChange)
}

export function getSnapshot(): OutboxState {
  return state
}

const kosong: OutboxState = { items: [], failures: [], flushing: false }
export function getServerSnapshot(): OutboxState {
  return kosong
}

export function enqueue(item: Omit<OutboxItem, "attempts">) {
  // Kiriman untuk penanda yang sama tidak pernah digandakan di antrean. Ini
  // pertahanan pertama; server tetap punya pertahanan keduanya lewat unique index.
  if (state.items.some((i) => i.clientEventId === item.clientEventId)) return

  const items = [...state.items, { ...item, attempts: 0 }]
  persist(items)
  publish({ ...state, items })

  void flush()
}

export function dismissFailures() {
  publish({ ...state, failures: [] })
}

/** Mengirimkan antrean secara berurutan.
 *
 *  Berurutan, bukan paralel: laporan dari satu inspektor untuk satu order harus
 *  sampai dalam urutan ia menekannya. Mengirim bersamaan membuat server menerima
 *  "Selesai" sebelum "Mulai", yang akan ditolak sebagai keluar urutan padahal
 *  keduanya sah. */
export async function flush(): Promise<void> {
  if (state.flushing || state.items.length === 0) return
  publish({ ...state, flushing: true })

  const antrian = [...state.items]
  const terkirim = new Set<string>()
  const gagal: OutboxState["failures"] = []

  for (const item of antrian) {
    try {
      const hasil = await submitStatusEvent(item.actorId, item.orderId, {
        to_status: item.toStatus,
        client_event_id: item.clientEventId,
        occurred_at: item.occurredAt,
        reason: item.reason,
      })

      // Layar berubah seketika tanpa menunggu pesan real-time yang membawa
      // perubahan yang sama kembali.
      apply(hasil.order)
      terkirim.add(item.clientEventId)

      // Laporan yang ditolak tetap keluar dari antrean: server sudah
      // mencatatnya, jadi mengirim ulang tidak akan mengubah apa pun. Yang
      // ditolak adalah perubahan statusnya, bukan pengirimannya (B-07).
      if (!hasil.accepted && !hasil.duplicate) {
        gagal.push({ orderRef: item.orderRef, message: hasil.message })
      }
    } catch (error) {
      if (error instanceof ApiError) {
        // Server menjawab dan menolak. Mengulanginya tidak akan berhasil.
        terkirim.add(item.clientEventId)
        gagal.push({ orderRef: item.orderRef, message: error.message })
        continue
      }
      // Gagal jaringan: laporan tetap di antrean dan dicoba lagi nanti. Sisa
      // antrean tidak dilanjutkan supaya urutannya tetap terjaga.
      break
    }
  }

  // WARNING: sisa dihitung dari state TERKINI, bukan dari salinan yang diambil
  // sebelum perulangan. Inspektor bisa menekan tombol lagi selagi pengiriman
  // berjalan, dan laporan itu sudah masuk antrean — memakai salinan lama akan
  // menghapusnya tanpa jejak.
  const sisa = state.items.filter((i) => !terkirim.has(i.clientEventId))

  persist(sisa)
  publish({
    items: sisa,
    failures: [...state.failures, ...gagal],
    flushing: false,
  })
}

const RETRY_INTERVAL = 15_000

/** Menghidupkan pengiriman ulang otomatis. Mengembalikan fungsi penghentiannya. */
export function startFlushing(): () => void {
  hydrate()
  void flush()

  const onOnline = () => void flush()
  window.addEventListener("online", onOnline)
  const timer = setInterval(() => void flush(), RETRY_INTERVAL)

  return () => {
    window.removeEventListener("online", onOnline)
    clearInterval(timer)
  }
}

/** Hanya untuk test. */
export function reset() {
  state = { items: [], failures: [], flushing: false }
  loaded = false
  listeners.clear()
  try {
    localStorage.removeItem(KEY)
  } catch {
    // Tidak ada penyimpanan; tidak ada yang perlu dibersihkan.
  }
}
