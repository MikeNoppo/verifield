"use client"

import { cn } from "@/lib/utils"
import { useConnection, useLastUpdate } from "@/lib/live/hooks"
import { waktu } from "@/lib/format"

const TEKS = {
  connecting: "Menyambung",
  online: "Tersambung",
  reconnecting: "Menyambung ulang",
  offline: "Terputus",
} as const

/** Layar yang diam punya dua arti yang tidak bisa dibedakan — pekerjaan memang
    belum berubah, atau sambungan terputus — dan perbedaan itulah yang menentukan
    klien perlu menelepon atau tidak (F-01).

    Statusnya dibaca dari koneksi stream yang sebenarnya, bukan dari
    navigator.onLine: jaringan bisa saja hidup sementara server tidak terjangkau,
    dan yang penting bagi pengguna adalah apakah kabar dari lapangan masih
    mengalir. */
export function ConnectionIndicator({
  compact = false,
  className,
}: {
  compact?: boolean
  className?: string
}) {
  const state = useConnection()
  const lastUpdate = useLastUpdate()

  return (
    <span
      className={cn("inline-flex items-center gap-2 text-xs", className)}
      role="status"
      aria-live="polite"
    >
      <span className="relative flex size-2 shrink-0 items-center justify-center">
        {state === "reconnecting" || state === "connecting" ? (
          <span className="absolute size-2 animate-ping rounded-full bg-attention/70" />
        ) : null}
        <span
          className={cn(
            "size-2 rounded-full",
            state === "online" && "bg-status-completed",
            (state === "reconnecting" || state === "connecting") && "bg-attention",
            state === "offline" && "bg-destructive",
          )}
        />
      </span>

      <span
        className={cn(
          state === "online" ? "text-muted-foreground" : "font-medium text-foreground",
        )}
      >
        {TEKS[state]}
      </span>

      {!compact && lastUpdate ? (
        <span className="tabular text-muted-foreground/70">
          · kabar terakhir {waktu(lastUpdate)}
        </span>
      ) : null}
    </span>
  )
}
