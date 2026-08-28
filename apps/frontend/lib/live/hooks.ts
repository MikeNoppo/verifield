"use client"

import * as React from "react"

import { listEvents } from "@/lib/api/orders"
import type { JobOrder, StatusEvent } from "@/lib/domain/types"
import {
  apply,
  getServerSnapshot,
  getSnapshot,
  subscribe,
  type ConnectionState,
  type LiveState,
} from "./store"

export function useLive(): LiveState {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}

export function useConnection(): ConnectionState {
  return useLive().connection
}

export function useLastUpdate(): string | null {
  return useLive().lastMessageAt
}

/** Menumpangkan pembaruan langsung di atas daftar hasil render server.
 *
 *  Urutan dan penyaringan tetap milik server — klien hanya mengganti isi baris
 *  yang sudah ada. Order yang benar-benar baru tidak disisipkan di sini karena
 *  itu akan melanggar saringan yang sedang aktif; store yang menangani kasus itu
 *  dengan memuat ulang halaman. */
export function useLiveList(initial: JobOrder[]): JobOrder[] {
  const live = useLive()

  return React.useMemo(
    () =>
      initial.map((order) => {
        const latest = live.orders.get(order.id)
        return latest && latest.seq > order.seq ? latest : order
      }),
    [initial, live],
  )
}

export function useLiveOrder(initial: JobOrder): JobOrder {
  const live = useLive()
  const latest = live.orders.get(initial.id)

  return React.useMemo(() => {
    if (!latest || latest.seq <= initial.seq) return initial
    // Snapshot real-time sengaja tidak membawa riwayat; riwayat yang sudah
    // dimiliki halaman dipertahankan dan dilengkapi useLiveEvents.
    return { ...latest, events: initial.events }
  }, [initial, latest])
}

/** Melengkapi timeline dengan entri yang muncul setelah halaman dirender.
 *
 *  Pesan real-time hanya membawa keadaan order, bukan riwayatnya — kalau riwayat
 *  ikut dikirim, seluruh timeline akan dikirim ulang ke setiap klien setiap kali
 *  satu entri bertambah. Di sini selisihnya diambil sekali per perubahan, hanya
 *  oleh layar yang memang sedang menampilkan riwayat. */
export function useLiveEvents(order: JobOrder, actorId: string): StatusEvent[] {
  const live = useLiveOrder(order)
  const [extra, setExtra] = React.useState<StatusEvent[]>([])

  // Batas seq yang sudah pernah diminta. Disimpan di ref supaya perubahannya
  // tidak memicu effect berikutnya — kalau tidak, setiap pengambilan akan
  // menjadwalkan pengambilan berikutnya tanpa henti.
  const requestedUpTo = React.useRef(0)

  React.useEffect(() => {
    const held = Math.max(
      0,
      ...order.events.map((e) => e.seq),
      ...extra.map((e) => e.seq),
    )
    if (live.seq <= held || live.seq <= requestedUpTo.current) return

    requestedUpTo.current = live.seq
    let cancelled = false

    listEvents(actorId, order.id, held)
      .then((fetched) => {
        if (cancelled || fetched.length === 0) return
        setExtra((previous) => {
          const seen = new Set(previous.map((e) => e.id))
          return [...previous, ...fetched.filter((e) => !seen.has(e.id))]
        })
      })
      .catch(() => {
        // Timeline akan menyusul pada perubahan berikutnya, atau saat halaman
        // dibuka ulang. Kegagalan di sini tidak layak mengganggu layar.
        requestedUpTo.current = 0
      })

    return () => {
      cancelled = true
    }
  }, [live.seq, order.id, order.events, extra, actorId])

  return React.useMemo(() => {
    if (extra.length === 0) return order.events
    const seen = new Set(order.events.map((e) => e.id))
    return [...order.events, ...extra.filter((e) => !seen.has(e.id))]
  }, [order.events, extra])
}

/** Menerapkan hasil mutasi ke store agar layar berubah seketika, tanpa menunggu
    pesan real-time yang membawa perubahan yang sama kembali. */
export function useApplyResult(): (order: JobOrder) => void {
  return React.useCallback((order: JobOrder) => {
    apply(order)
  }, [])
}
