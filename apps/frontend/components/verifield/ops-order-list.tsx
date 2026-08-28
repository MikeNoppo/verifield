"use client"

import Link from "next/link"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusRail } from "@/components/verifield/status-rail"
import { AssignDialog } from "@/components/verifield/assign-dialog"
import { AttentionCard, StatusDistributionBar } from "@/components/verifield/ops-widgets"
import { PageHeading } from "@/components/verifield/app-shell"
import { Card } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveList } from "@/lib/live/hooks"
import {
  attentionCounts,
  countRunning,
  statusDistribution,
  type AttentionKey,
} from "@/lib/domain/summary"
import { hasOpenAlert, isStale, needsAssignment } from "@/lib/domain/status"
import type { Inspector, JobOrder } from "@/lib/domain/types"
import { relativeTime } from "@/lib/format"

const ATTENTION = [
  {
    key: "unassigned",
    label: "Butuh penugasan",
    hint: "Order diterima, inspektor belum ditentukan",
  },
  {
    key: "cancellation",
    label: "Permintaan pembatalan",
    hint: "Diajukan setelah pekerjaan dimulai, menunggu keputusan Anda",
  },
  { key: "stale", label: "Tanpa pembaruan >8 jam", hint: "Tindak lanjuti sebelum klien bertanya" },
  {
    key: "late_update",
    label: "Perlu tindak lanjut",
    hint: "Ada pekerjaan nyata yang perlu diselesaikan kompensasinya",
  },
] as const

function cocok(order: JobOrder, key: AttentionKey | undefined, now: Date): boolean {
  if (!key) return true
  if (key === "unassigned") return needsAssignment(order)
  if (key === "cancellation") return order.cancellationRequested
  if (key === "stale") return isStale(order, now)
  return hasOpenAlert(order)
}

/** Penyaringan, penghitungan, dan distribusi dihitung dari satu daftar yang sama
    di klien. Dengan begitu semuanya bergerak bersama saat pembaruan masuk —
    angka pada kartu perhatian ikut turun begitu ordernya ditangani, tanpa
    memuat ulang halaman. */
export function OpsOrderList({
  orders: initial,
  inspectors,
  attention,
}: {
  orders: JobOrder[]
  inspectors: Inspector[]
  attention?: AttentionKey
}) {
  const semua = useLiveList(initial)
  const now = new Date()
  const href = useActorHref()
  const counts = attentionCounts(semua, now)
  const orders = semua.filter((o) => cocok(o, attention, now))

  return (
    <>
      <PageHeading
        title="Order Aktif"
        meta={
          <>
            {countRunning(semua)} berjalan · {semua.length} total
          </>
        }
      />

      <div className="mb-4 grid gap-2.5 sm:grid-cols-2 xl:grid-cols-4">
        {ATTENTION.map((item) => (
          <AttentionCard
            key={item.key}
            label={item.label}
            hint={item.hint}
            count={counts[item.key]}
            href={href(attention === item.key ? "/ops" : `/ops?a=${item.key}`)}
            active={attention === item.key}
          />
        ))}
      </div>

      <Card className="mb-4 p-4">
        <StatusDistributionBar data={statusDistribution(semua)} />
      </Card>

      {attention ? (
        <div className="mb-3 flex items-center gap-2 text-xs">
          <span className="text-muted-foreground">
            Disaring: {ATTENTION.find((x) => x.key === attention)?.label}
          </span>
          <Link href={href("/ops")} className="font-medium underline underline-offset-2">
            Hapus saringan
          </Link>
        </div>
      ) : null}

      {orders.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Tidak ada order pada saringan ini</EmptyTitle>
            <EmptyDescription>
              Semua tertangani. Hapus saringan untuk melihat seluruh order.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[112px]">Ref</TableHead>
                <TableHead className="w-[200px]">Klien</TableHead>
                <TableHead className="min-w-[220px]">Objek &amp; Lokasi</TableHead>
                <TableHead className="w-[148px]">Inspektor</TableHead>
                <TableHead className="w-[236px]">Status</TableHead>
                <TableHead className="w-[124px] text-right">Diperbarui</TableHead>
                <TableHead className="w-[112px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((o) => {
                const basi = isStale(o, now)
                const perluTindakan = basi || hasOpenAlert(o) || o.cancellationRequested

                return (
                  <TableRow key={o.id} className="h-10">
                    <TableCell className="relative align-middle">
                      {/* Penanda urgensi hidup di kolom paling kiri, bukan di
                          warna status — dua sistem yang berbeda. */}
                      {perluTindakan ? (
                        <span className="absolute inset-y-1 left-0 w-0.5 rounded-full bg-attention" />
                      ) : null}
                      <Link
                        href={href(`/ops/order/${o.id}`)}
                        prefetch={false}
                        className="tabular font-mono text-xs font-medium hover:underline"
                      >
                        {o.ref}
                      </Link>
                    </TableCell>

                    <TableCell className="truncate align-middle text-xs">{o.clientName}</TableCell>

                    <TableCell className="align-middle">
                      <div className="flex flex-col gap-0.5">
                        <span className="truncate text-xs">{o.object}</span>
                        <span className="truncate text-[11px] text-muted-foreground">
                          {o.location}
                        </span>
                      </div>
                    </TableCell>

                    <TableCell className="align-middle text-xs">
                      {o.inspectorName ?? <span className="text-muted-foreground">Belum ada</span>}
                    </TableCell>

                    <TableCell className="align-middle">
                      <div className="flex items-center gap-3">
                        <StatusRail order={o} className="w-20 shrink-0" />
                        <StatusBadge status={o.status} size="sm" />
                      </div>
                    </TableCell>

                    <TableCell
                      className={cn(
                        "tabular text-right align-middle text-xs",
                        basi ? "font-medium text-attention" : "text-muted-foreground",
                      )}
                    >
                      {relativeTime(o.statusChangedAt, now)}
                    </TableCell>

                    <TableCell className="align-middle text-right">
                      {needsAssignment(o) ? (
                        <AssignDialog order={o} inspectors={inspectors} compact />
                      ) : o.cancellationRequested ? (
                        <Link
                          href={href(`/ops/order/${o.id}`)}
                          prefetch={false}
                          className="text-xs font-medium text-attention hover:underline"
                        >
                          Tinjau
                        </Link>
                      ) : null}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  )
}
