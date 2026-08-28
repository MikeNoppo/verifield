import { CloudOffIcon, PencilLineIcon, TriangleAlertIcon } from "lucide-react"

import { cn } from "@/lib/utils"
import { STATUS_LABEL } from "@/lib/domain/status"
import type { StatusEvent, Status } from "@/lib/domain/types"
import { DualTimestamp } from "./dual-timestamp"
import { tanggalLengkap } from "@/lib/format"

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

const ROLE_LABEL = {
  klien: "Klien",
  ops: "Koordinator",
  lapangan: "Inspektor",
} as const

function judul(e: StatusEvent): string {
  if (e.kind === "correction") {
    return `Koreksi ke ${STATUS_LABEL[e.to]}`
  }
  if (e.kind === "cancellation_request") return "Pembatalan diajukan"
  if (e.kind === "late_rejected") return `Laporan ${STATUS_LABEL[e.to]} ditolak`
  return STATUS_LABEL[e.to]
}

function Entry({ event, showAudit }: { event: StatusEvent; showAudit: boolean }) {
  const rejected = event.kind === "late_rejected"
  const correction = event.kind === "correction"
  const request = event.kind === "cancellation_request"

  return (
    <li className="relative flex gap-4 pb-6 last:pb-0">
      <span className="absolute top-3 bottom-0 left-[5px] w-px bg-border last:hidden" />

      <span
        className={cn(
          "relative mt-1.5 flex size-2.5 shrink-0 items-center justify-center rounded-full",
          rejected && "bg-transparent ring-2 ring-muted-foreground/40",
          correction && "bg-transparent ring-2 ring-attention/70",
          request && "bg-transparent ring-2 ring-attention/70",
          !rejected && !correction && !request && DOT[event.to],
        )}
      />

      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <span className="flex items-center gap-2">
            <span
              className={cn(
                "text-sm font-medium",
                rejected ? "text-muted-foreground line-through" : "text-foreground",
              )}
            >
              {judul(event)}
            </span>
            {rejected ? <CloudOffIcon className="size-3.5 text-muted-foreground" /> : null}
            {correction ? <PencilLineIcon className="size-3.5 text-attention" /> : null}
            {request ? <TriangleAlertIcon className="size-3.5 text-attention" /> : null}
          </span>

          <DualTimestamp
            eventTime={event.eventTime}
            receivedTime={event.receivedTime}
            withDate
            className="items-end text-right"
          />
        </div>

        <span className="text-xs text-muted-foreground">
          {ROLE_LABEL[event.actorRole]} · {event.actorName}
        </span>

        {event.reason ? (
          <p
            className={cn(
              "rounded-md border px-3 py-2 text-xs leading-relaxed",
              rejected
                ? "border-border bg-muted/40 text-muted-foreground"
                : correction || request
                  ? "border-attention/25 bg-attention/8 text-foreground"
                  : "border-border bg-muted/40 text-muted-foreground",
            )}
          >
            {event.reason}
          </p>
        ) : null}

        {showAudit ? (
          <span className="tabular font-mono text-[10px] text-muted-foreground/70">
            {event.idempotencyKey} · diterima {tanggalLengkap(event.receivedTime)}
          </span>
        ) : null}
      </div>
    </li>
  )
}

/** Urutan mengikuti waktu kejadian, bukan waktu terima. Empat pembaruan yang
    tiba sekaligus setelah sinyal pulih tetap tampil dalam urutan yang benar
    bagi klien (B-02, B-06). */
export function EventTimeline({
  events,
  showAudit = false,
}: {
  events: StatusEvent[]
  showAudit?: boolean
}) {
  const urut = [...events].sort(
    (a, b) => new Date(a.eventTime).getTime() - new Date(b.eventTime).getTime(),
  )

  return (
    <ol className="flex flex-col">
      {urut.map((e) => (
        <Entry key={e.id} event={e} showAudit={showAudit} />
      ))}
    </ol>
  )
}
