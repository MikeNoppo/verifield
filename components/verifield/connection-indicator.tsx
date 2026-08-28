"use client"

import { cn } from "@/lib/utils"
import { useConnection } from "@/hooks/use-connection"

const TEKS = {
  online: "Tersambung",
  reconnecting: "Menyambung ulang",
  offline: "Terputus",
} as const

export function ConnectionIndicator({
  lastUpdate,
  className,
}: {
  /** Jam pembaruan terakhir, sudah diformat di server. Ditampilkan permanen,
      bukan hanya saat putus — supaya klien selalu tahu seberapa mutakhir
      yang sedang dilihatnya. */
  lastUpdate: string
  className?: string
}) {
  const state = useConnection()

  return (
    <span
      className={cn("inline-flex items-center gap-2 text-xs", className)}
      role="status"
      aria-live="polite"
    >
      <span className="relative flex size-2 shrink-0 items-center justify-center">
        {state === "reconnecting" ? (
          <span className="absolute size-2 animate-ping rounded-full bg-attention/70" />
        ) : null}
        <span
          className={cn(
            "size-2 rounded-full",
            state === "online" && "bg-status-completed",
            state === "reconnecting" && "bg-attention",
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

      <span className="tabular text-muted-foreground/70">· diperbarui {lastUpdate}</span>
    </span>
  )
}
