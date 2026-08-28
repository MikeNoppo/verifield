import Link from "next/link"
import type { Route } from "next"

import { cn } from "@/lib/utils"
import { STATUS_LABEL } from "@/lib/domain/status"
import type { Status } from "@/lib/domain/types"

/** Urgensi sengaja dipisahkan dari status. Status menjawab "di tahap mana",
    urgensi menjawab "butuh saya sekarang atau tidak" — mencampurnya memaksa
    koordinator menghafal warna mana yang berarti kerjakan-saya. */
export function AttentionCard({
  label,
  count,
  hint,
  href,
  active,
}: {
  label: string
  count: number
  hint: string
  href: Route
  active: boolean
}) {
  const urgent = count > 0

  return (
    <Link
      href={href}
      className={cn(
        "group flex flex-col gap-1 rounded-xl border p-3 transition-colors",
        active
          ? "border-foreground/25 bg-muted/60"
          : urgent
            ? "border-attention/25 bg-attention/6 hover:bg-attention/10"
            : "border-border hover:bg-muted/40",
      )}
    >
      <span className="flex items-baseline gap-2">
        <span
          className={cn(
            "tabular text-xl leading-none font-semibold",
            urgent ? "text-attention" : "text-muted-foreground",
          )}
        >
          {count}
        </span>
        <span className="text-xs font-medium">{label}</span>
      </span>
      <span className="text-[11px] leading-tight text-muted-foreground">{hint}</span>
    </Link>
  )
}

const FILL: Record<Status, string> = {
  requested: "var(--status-requested)",
  assigned: "var(--status-assigned)",
  on_the_way: "var(--status-on-the-way)",
  on_site: "var(--status-on-site)",
  in_progress: "var(--status-in-progress)",
  completed: "var(--status-completed)",
  failed: "var(--status-failed)",
  cancelled: "var(--status-cancelled)",
}

/** Satu batang bertumpuk untuk seluruh order. Ringkasan sepadat mungkin yang
    memakai warna status untuk sesuatu yang benar-benar berarti. */
export function StatusDistributionBar({
  data,
}: {
  data: Array<{ status: Status; count: number }>
}) {
  const total = data.reduce((s, d) => s + d.count, 0)
  if (total === 0) return null

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex h-2 w-full overflow-hidden rounded-4xl">
        {data.map((d) => (
          <div
            key={d.status}
            style={
              {
                "--seg": FILL[d.status],
                width: `${(d.count / total) * 100}%`,
              } as React.CSSProperties
            }
            className="h-full bg-(--seg) first:rounded-l-4xl last:rounded-r-4xl"
            title={`${STATUS_LABEL[d.status]}: ${d.count}`}
          />
        ))}
      </div>

      <ul className="flex flex-wrap gap-x-4 gap-y-1.5">
        {data.map((d) => (
          <li key={d.status} className="flex items-center gap-1.5 text-[11px]">
            <span
              style={{ "--seg": FILL[d.status] } as React.CSSProperties}
              className="size-2 shrink-0 rounded-full bg-(--seg)"
            />
            <span className="text-muted-foreground">{STATUS_LABEL[d.status]}</span>
            <span className="tabular font-medium">{d.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
