"use client"

import * as React from "react"

import {
  getServerSnapshot,
  getSnapshot,
  startFlushing,
  subscribe,
  type OutboxState,
} from "./queue"

export function useOutbox(): OutboxState {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}

/** Menghidupkan pengiriman ulang otomatis selama layar inspektor terbuka.
 *
 *  Dipasang di layout, bukan di tiap halaman, supaya antrean tetap terkirim
 *  ketika inspektor berpindah antar order — bahkan ketika ia meninggalkan
 *  halaman order yang laporannya sedang menunggu. */
export function useOutboxFlusher() {
  React.useEffect(() => startFlushing(), [])
}
