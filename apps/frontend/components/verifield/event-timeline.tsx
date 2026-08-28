import {
  ClockAlertIcon,
  CloudOffIcon,
  HandshakeIcon,
  PencilLineIcon,
  RotateCcwIcon,
  SkipForwardIcon,
  TriangleAlertIcon,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { STATUS_LABEL } from "@/lib/domain/status"
import type { EventKind, Role, Status, StatusEvent } from "@/lib/domain/types"
import { DualTimestamp } from "./dual-timestamp"
import { formatFullDateTime } from "@/lib/format"

const DOT: Record<Status, string> = {
  requested: "bg-status-requested",
  assigned: "bg-status-assigned",
  on_the_way: "bg-status-on-the-way",
  on_site: "bg-status-on-site",
  in_progress: "bg-status-in-progress",
  completed: "bg-status-completed",
  failed: "bg-status-failed",
  cancelled: "bg-status-cancelled",
}

const ROLE_LABEL: Record<Role, string> = {
  client: "Klien",
  admin: "Koordinator",
  inspector: "Inspektor",
  cs: "Customer Service",
}

/** Entri yang tidak mengubah status digambar berbeda dari entri yang mengubah.
    Perbedaannya penting: klien yang melihat "Selesai" dicoret harus langsung
    paham bahwa pekerjaannya TIDAK berpindah ke Selesai. */
const APPLIED: Record<EventKind, boolean> = {
  transition: true,
  correction: true,
  late_rejected: false,
  out_of_order: false,
  skipped_step: false,
  cancellation_request: false,
  cancellation_rejected: false,
  settlement_pending: false,
  settlement_decided: false,
  rejected_other: false,
}

const ICON: Partial<Record<EventKind, typeof CloudOffIcon>> = {
  correction: PencilLineIcon,
  late_rejected: CloudOffIcon,
  out_of_order: ClockAlertIcon,
  skipped_step: SkipForwardIcon,
  cancellation_request: TriangleAlertIcon,
  cancellation_rejected: RotateCcwIcon,
  settlement_pending: TriangleAlertIcon,
  settlement_decided: HandshakeIcon,
  rejected_other: ClockAlertIcon,
}

function title(e: StatusEvent): string {
  switch (e.kind) {
    case "correction":
      return `Koreksi ke ${STATUS_LABEL[e.to]}`
    case "cancellation_request":
      return "Pembatalan diajukan"
    case "cancellation_rejected":
      return "Permintaan pembatalan ditolak"
    case "settlement_pending":
      return "Pembatalan tidak lagi dapat diterapkan — menunggu penyelesaian"
    case "settlement_decided":
      return "Penyelesaian komersial diputuskan"
    case "rejected_other":
      return `Laporan ${STATUS_LABEL[e.to]} tidak diterapkan`
    case "late_rejected":
      return `Laporan ${STATUS_LABEL[e.to]} ditolak — order sudah ditutup`
    case "out_of_order":
      return `Laporan ${STATUS_LABEL[e.to]} ditolak — status sudah pada tahap ini atau sesudahnya`
    case "skipped_step":
      return `Laporan ${STATUS_LABEL[e.to]} ditolak — ada tahap yang belum dilaporkan`
    default:
      return STATUS_LABEL[e.to]
  }
}

function Entry({ event, showAudit }: { event: StatusEvent; showAudit: boolean }) {
  const applied = APPLIED[event.kind]
  const Icon = ICON[event.kind]
  const isCorrection = event.kind === "correction"

  return (
    <li className="relative flex gap-4 pb-6 last:pb-0">
      <span className="absolute top-3 bottom-0 left-[5px] w-px bg-border last:hidden" />

      <span
        className={cn(
          "relative mt-1.5 flex size-2.5 shrink-0 items-center justify-center rounded-full",
          !applied && "bg-transparent ring-2 ring-muted-foreground/40",
          isCorrection && "bg-transparent ring-2 ring-attention/70",
          applied && !isCorrection && DOT[event.to],
        )}
      />

      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <span className="flex items-center gap-2">
            <span
              className={cn(
                "text-sm font-medium",
                applied ? "text-foreground" : "text-muted-foreground line-through",
              )}
            >
              {title(event)}
            </span>
            {Icon ? (
              <Icon
                className={cn(
                  "size-3.5",
                  isCorrection || event.kind === "cancellation_request"
                    ? "text-attention"
                    : "text-muted-foreground",
                )}
              />
            ) : null}
          </span>

          <DualTimestamp
            eventTime={event.eventTime}
            receivedTime={event.receivedTime}
            withDate
            className="items-end text-right"
          />
        </div>

        <span className="text-xs text-muted-foreground">
          {ROLE_LABEL[event.actorRole]}
          {event.actorName ? ` · ${event.actorName}` : ""}
        </span>

        {event.reason ? (
          <p
            className={cn(
              "rounded-md border px-3 py-2 text-xs leading-relaxed",
              isCorrection || event.kind === "cancellation_request"
                ? "border-attention/25 bg-attention/8 text-foreground"
                : "border-border bg-muted/40 text-muted-foreground",
            )}
          >
            {event.reason}
          </p>
        ) : null}

        {/* Jam perangkat di luar batas wajar. Ditampilkan karena waktu kejadian
            adalah dasar laporan dan penagihan — pembacanya berhak tahu bahwa
            angka yang dilihatnya berasal dari server, bukan dari lapangan (B-02). */}
        {event.timeAdjusted ? (
          <span className="text-[11px] leading-relaxed text-muted-foreground">
            Waktu yang dilaporkan perangkat berada di luar batas wajar, sehingga waktu
            penerimaan sistem yang dipakai.
          </span>
        ) : null}

        {showAudit ? (
          <span className="tabular font-mono text-[10px] text-muted-foreground/70">
            #{event.seq}
            {event.idempotencyKey ? ` · ${event.idempotencyKey}` : ""} · diterima{" "}
            {formatFullDateTime(event.receivedTime)}
          </span>
        ) : null}
      </div>
    </li>
  )
}

/** Urutan mengikuti waktu kejadian, bukan waktu terima. Empat pembaruan yang
    tiba sekaligus setelah sinyal pulih tetap tampil dalam urutan yang benar
    bagi klien (B-02, B-06). Seq jadi pemecah seri agar urutannya stabil. */
export function EventTimeline({
  events,
  showAudit = false,
}: {
  events: StatusEvent[]
  showAudit?: boolean
}) {
  const ordered = [...events].sort((a, b) => {
    const millisBetween = new Date(a.eventTime).getTime() - new Date(b.eventTime).getTime()
    return millisBetween !== 0 ? millisBetween : a.seq - b.seq
  })

  return (
    <ol className="flex flex-col">
      {ordered.map((e) => (
        <Entry key={e.id} event={e} showAudit={showAudit} />
      ))}
    </ol>
  )
}
