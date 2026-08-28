"use client"

import Link from "next/link"
import { ChevronRightIcon, MapPinIcon } from "lucide-react"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusRail } from "@/components/verifield/status-rail"
import { SyncQueueBanner } from "@/components/verifield/next-action"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveList } from "@/lib/live/hooks"
import { inspectorActions, isTerminal } from "@/lib/domain/status"
import type { JobOrder } from "@/lib/domain/types"
import { formatDateTime } from "@/lib/format"

export function LapanganTaskList({ orders: initial }: { orders: JobOrder[] }) {
  const semua = useLiveList(initial)
  const href = useActorHref()
  // Order yang baru saja selesai hilang dari daftar tugas begitu laporannya
  // terkirim — tanpa memuat ulang halaman.
  const tasks = semua.filter((o) => !isTerminal(o.status))

  return (
    <>
      <h1 className="mb-1 text-lg font-semibold tracking-tight">Tugas Saya</h1>
      <p className="mb-4 text-xs text-muted-foreground">{tasks.length} penugasan berjalan</p>

      <SyncQueueBanner />

      {tasks.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Tidak ada penugasan berjalan</EmptyTitle>
            <EmptyDescription>Koordinator akan menugaskan pekerjaan next.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ul className="flex flex-col gap-3">
          {tasks.map((o) => {
            const next = inspectorActions(o.status).find((a) => a.tone === "primary")

            return (
              <li key={o.id}>
                <Link
                  href={href(`/lapangan/order/${o.id}`)}
                  prefetch={false}
                  // Target sentuh besar: dipakai sambil berdiri, sering
                  // mengenakan sarung tangan.
                  className="flex min-h-14 flex-col gap-3 rounded-xl border border-border bg-card p-4 transition-colors active:bg-muted/60"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 flex-col gap-1">
                      <span className="tabular font-mono text-xs text-muted-foreground">
                        {o.ref}
                      </span>
                      <span className="text-sm leading-snug font-medium">{o.object}</span>
                      <span className="flex items-center gap-1 text-xs text-muted-foreground">
                        <MapPinIcon className="size-3 shrink-0" />
                        {o.location}
                      </span>
                    </div>
                    <StatusBadge status={o.status} size="sm" />
                  </div>

                  <StatusRail order={o} className="w-full" />

                  <div className="flex items-center justify-between gap-2 border-t border-border pt-3">
                    <span className="tabular text-xs text-muted-foreground">
                      {formatDateTime(o.scheduledAt)}
                    </span>
                    {next ? (
                      <span className="flex items-center gap-1 text-sm font-semibold">
                        {next.label}
                        <ChevronRightIcon className="size-4" />
                      </span>
                    ) : null}
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </>
  )
}
