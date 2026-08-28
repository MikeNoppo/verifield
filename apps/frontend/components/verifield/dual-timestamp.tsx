import { cn } from "@/lib/utils"
import { durasi, selisih, tanggal, waktu } from "@/lib/format"

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
  const lag = selisih(eventTime, receivedTime)
  const tertunda = lag >= 60_000

  return (
    <span className={cn("flex flex-col gap-0.5", className)}>
      <span className="tabular font-mono text-sm text-foreground">
        {withDate ? `${tanggal(eventTime)} ` : ""}
        {waktu(eventTime)}
      </span>
      {tertunda ? (
        <span className="tabular text-[11px] leading-tight text-muted-foreground">
          dilaporkan {waktu(receivedTime)} · tertunda {durasi(lag)}
        </span>
      ) : null}
    </span>
  )
}
