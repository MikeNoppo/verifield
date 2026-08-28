"use client"

import { useOutboxFlusher } from "@/lib/offline/hooks"

/** Menjaga antrean laporan tetap terkirim selama layar inspektor terbuka,
    termasuk saat ia berpindah antar order. */
export function OutboxProvider({ children }: { children: React.ReactNode }) {
  useOutboxFlusher()
  return children
}
