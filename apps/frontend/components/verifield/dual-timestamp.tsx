import { cn } from "@/lib/utils"
import { formatDuration, millisBetween, formatDate, formatTime } from "@/lib/format"

/** Waktu kejadian di lapangan ditampilkan menonjol karena itulah yang sah untuk
    laporan dan penagihan. Waktu terima tetap ikut, tetapi sebagai jejak — dan
    hanya muncul ketika keduanya benar-benar berbeda (B-02). */
export function DualTimestamp({
  eventTime,
  receivedTime,
  withDate = false,
  className,
}: {
  eventTime: string
  receivedTime: string
  withDate?: boolean
  className?: string
}) {
  const lag = millisBetween(eventTime, receivedTime)
  const tertunda = lag >= 60_000

  return (
    <span className={cn("flex flex-col gap-0.5", className)}>
      <span className="tabular font-mono text-sm text-foreground">
        {withDate ? `${formatDate(eventTime)} ` : ""}
        {formatTime(eventTime)}
      </span>
      {tertunda ? (
        <span className="tabular text-[11px] leading-tight text-muted-foreground">
          dilaporkan {formatTime(receivedTime)} · tertunda {formatDuration(lag)}
        </span>
      ) : null}
    </span>
  )
}
