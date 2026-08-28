import { BanIcon, XIcon } from "lucide-react"

import { cn } from "@/lib/utils"
import { PIPELINE, STATUS_LABEL, isTerminal, pipelineIndex } from "@/lib/domain/status"
import type { JobOrder } from "@/lib/domain/types"

/** Warna tiap ruas mengambil langkah ramp-nya sendiri, sehingga rel ini
    memperlihatkan intensitas yang menaik — bukan enam kotak berwarna sama.
    Nilainya disuntik sebagai custom property, bukan nama kelas, karena nama
    kelas dinamis tidak selamat dari pemindaian Tailwind. */
const RAMP: Record<(typeof PIPELINE)[number], string> = {
  requested: "var(--status-requested)",
  assigned: "var(--status-assigned)",
  on_the_way: "var(--status-on-the-way)",
  on_site: "var(--status-on-site)",
  in_progress: "var(--status-in-progress)",
  completed: "var(--status-completed)",
}

function seg(color: string) {
  return { "--seg": color } as React.CSSProperties
}

type Props = {
  order: Pick<JobOrder, "status" | "events">
  /** Di dalam tabel, StatusBadge di sebelahnya sudah membawa nama yang dapat
      diakses — rel cukup disembunyikan agar pembaca layar tidak mendengar
      enam ruas kosong per baris. */
  label?: "none" | "sr"
  className?: string
}

export function StatusRail({ order, label = "none", className }: Props) {
  const status = order.status
  const terminatedEarly = status === "failed" || status === "cancelled"

  // Tahap tempat pekerjaan berhenti, dibaca dari transisi terakhir yang masih
  // berada di dalam rel.
  const lastInRail = [...order.events]
    .reverse()
    .find((e) => e.kind === "transition" && !isTerminal(e.to))
  const exitAt = lastInRail ? pipelineIndex(lastInRail.to) : 0
  const currentAt = terminatedEarly ? exitAt : pipelineIndex(status)

  const visible = terminatedEarly ? PIPELINE.slice(0, exitAt + 1) : PIPELINE

  const srText = terminatedEarly
    ? `${STATUS_LABEL[status]} pada tahap ${STATUS_LABEL[PIPELINE[exitAt]!]}`
    : `${STATUS_LABEL[status]} — tahap ${currentAt + 1} dari ${PIPELINE.length}`

  return (
    <span
      className={cn("inline-flex h-2.5 w-full min-w-16 items-center gap-[2px]", className)}
      aria-hidden={label === "none" ? true : undefined}
      role={label === "sr" ? "img" : undefined}
      aria-label={label === "sr" ? srText : undefined}
    >
      {visible.map((step, i) => {
        const done = i < currentAt
        const current = i === currentAt

        return (
          <span
            key={step}
            style={done || current ? seg(RAMP[step]) : undefined}
            className={cn(
              "min-w-0 flex-1 rounded-[1px] transition-all",
              // Tinggi adalah kanal kedua di samping warna: ruas berjalan
              // lebih tebal, ruas yang tak akan pernah terjadi menipis jadi
              // rambut. Dibaca juga oleh mata yang tidak membedakan warna.
              current && "h-2.5 bg-(--seg)",
              done && "h-1.5 bg-(--seg)",
              !done && !current && "h-1 bg-muted-foreground/20",
            )}
          />
        )
      })}

      {terminatedEarly ? (
        // Terminus lebar bergaris putus memutus irama ruas yang teratur. Patahan
        // irama itu sinyal non-warna yang paling keras untuk "berhenti di sini,
        // sisanya tidak pernah terjadi".
        <span
          className={cn(
            "flex h-2.5 flex-[2] min-w-0 items-center justify-center rounded-[2px] border border-dashed",
            status === "failed"
              ? "border-status-failed/60 text-status-failed"
              : "border-status-cancelled/60 text-status-cancelled",
          )}
        >
          {status === "failed" ? (
            <XIcon className="size-2" strokeWidth={3} />
          ) : (
            <BanIcon className="size-2" strokeWidth={3} />
          )}
        </span>
      ) : null}
    </span>
  )
}

/** Versi bersemantik untuk halaman detail, tempat satu stepper per halaman
    membuat biaya markup-nya sepadan. */
export function StatusStepper({ order }: { order: Pick<JobOrder, "status" | "events"> }) {
  const status = order.status
  const terminatedEarly = status === "failed" || status === "cancelled"
  const lastInRail = [...order.events]
    .reverse()
    .find((e) => e.kind === "transition" && !isTerminal(e.to))
  const exitAt = lastInRail ? pipelineIndex(lastInRail.to) : 0
  const currentAt = terminatedEarly ? exitAt : pipelineIndex(status)

  return (
    <ol className="flex flex-col gap-0">
      {PIPELINE.map((step, i) => {
        const done = i < currentAt
        const current = i === currentAt
        const voided = terminatedEarly && i > exitAt
        const upcoming = !done && !current && !voided

        return (
          <li
            key={step}
            aria-current={current ? "step" : undefined}
            className="flex items-start gap-3"
          >
            <span className="flex flex-col items-center self-stretch">
              <span
                style={done || current ? seg(RAMP[step]) : undefined}
                className={cn(
                  "mt-1 size-2.5 shrink-0 rounded-full",
                  (done || current) && "bg-(--seg)",
                  current && "ring-3 ring-(--seg)/25",
                  upcoming && "bg-muted-foreground/25",
                  voided && "bg-muted-foreground/15",
                )}
              />
              {i < PIPELINE.length - 1 ? (
                <span
                  className={cn(
                    "w-px flex-1 min-h-5",
                    done ? "bg-muted-foreground/35" : "bg-muted-foreground/15",
                  )}
                />
              ) : null}
            </span>
            <span
              className={cn(
                "pb-4 text-sm",
                current && "font-medium text-foreground",
                done && "text-muted-foreground",
                (upcoming || voided) && "text-muted-foreground/60",
                voided && "line-through decoration-muted-foreground/40",
              )}
            >
              {STATUS_LABEL[step]}
            </span>
          </li>
        )
      })}

      {terminatedEarly ? (
        <li className="flex items-start gap-3">
          <span
            className={cn(
              "mt-0.5 flex size-2.5 shrink-0 items-center justify-center",
              status === "failed" ? "text-status-failed" : "text-status-cancelled",
            )}
          >
            {status === "failed" ? (
              <XIcon className="size-2.5" strokeWidth={3} />
            ) : (
              <BanIcon className="size-2.5" strokeWidth={3} />
            )}
          </span>
          <span
            className={cn(
              "text-sm font-medium",
              status === "failed" ? "text-status-failed" : "text-status-cancelled",
            )}
          >
            {STATUS_LABEL[status]}
          </span>
        </li>
      ) : null}
    </ol>
  )
}
