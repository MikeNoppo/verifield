"use client"

import * as React from "react"
import { useRouter } from "next/navigation"

import { connect, markOffline } from "@/lib/live/store"

/** Membuka satu stream perubahan untuk seluruh layar.
 *
 *  Dipasang di layout, bukan di tiap halaman, supaya berpindah antar halaman
 *  tidak memutus lalu membuka koneksi berulang kali — setiap pembukaan ulang
 *  berarti satu putaran replay yang sebenarnya tidak perlu. */
export function LiveProvider({
  actorId,
  children,
}: {
  actorId: string
  children: React.ReactNode
}) {
  const router = useRouter()

  React.useEffect(() => {
    // Order yang belum pernah terlihat tidak bisa disisipkan di klien tanpa
    // melanggar saringan yang sedang aktif di server, jadi halaman dimuat ulang.
    // router.refresh() hanya mengambil ulang komponen server — state klien,
    // posisi gulir, dan koneksi stream tetap utuh.
    return connect(actorId, () => router.refresh())
  }, [actorId, router])

  React.useEffect(() => {
    // Browser mengetahui jaringan hilang lebih cepat daripada EventSource
    // kehabisan waktu. Indikator koneksi ada justru untuk menghapus ambiguitas
    // "layar diam" (F-01), jadi kecepatan tahu itu berarti.
    window.addEventListener("offline", markOffline)
    return () => window.removeEventListener("offline", markOffline)
  }, [])

  return children
}
