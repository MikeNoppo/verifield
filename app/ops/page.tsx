import Link from "next/link"
import type { Route } from "next"

import { PageHeading } from "@/components/verifield/app-shell"
import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusRail } from "@/components/verifield/status-rail"
import { AssignDialog } from "@/components/verifield/assign-dialog"
import { AttentionCard, StatusDistributionBar } from "@/components/verifield/ops-widgets"
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
import {
  attentionCounts,
  listInspectors,
  listOrders,
  statusDistribution,
  type OrderFilter,
} from "@/lib/api/orders"
import { hasLateRejected, isStale, isTerminal } from "@/lib/domain/status"
import { DEMO_NOW } from "@/lib/demo-time"
import { sejak } from "@/lib/format"

const ATTENTION = [
  {
    key: "penugasan",
    label: "Butuh penugasan",
    hint: "Order diterima, inspektor belum ditentukan",
  },
  {
    key: "pembatalan",
    label: "Permintaan pembatalan",
    hint: "Diajukan setelah pekerjaan dimulai, menunggu keputusan Anda",
  },
  { key: "basi", label: "Tanpa pembaruan >8 jam", hint: "Tindak lanjuti sebelum klien bertanya" },
  {
    key: "terlambat",
    label: "Pembaruan terlambat ditolak",
    hint: "Ada pekerjaan nyata yang perlu diselesaikan kompensasinya",
  },
] as const

function first(v: string | string[] | undefined): string {
  return Array.isArray(v) ? (v[0] ?? "") : (v ?? "")
}

export default async function OpsDashboard({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const a = first((await searchParams).a)
  const attention = ATTENTION.find((x) => x.key === a)?.key
  const orders = listOrders({ attention } as OrderFilter)
  const counts = attentionCounts()
  const inspectors = listInspectors()

  return (
    <>
      <PageHeading
        title="Order Aktif"
        meta={
          <>
            {listOrders().filter((o) => !isTerminal(o.status)).length} berjalan ·{" "}
            {listOrders().length} total
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
            href={(attention === item.key ? "/ops" : `/ops?a=${item.key}`) as Route}
            active={attention === item.key}
          />
        ))}
      </div>

      <Card className="mb-4 p-4">
        <StatusDistributionBar data={statusDistribution()} />
      </Card>

      {attention ? (
        <div className="mb-3 flex items-center gap-2 text-xs">
          <span className="text-muted-foreground">
            Disaring: {ATTENTION.find((x) => x.key === attention)?.label}
          </span>
          <Link href="/ops" className="font-medium underline underline-offset-2">
            Hapus saringan
          </Link>
        </div>
      ) : null}

      {orders.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Tidak ada order pada saringan ini</EmptyTitle>
            <EmptyDescription>Semua tertangani. Hapus saringan untuk melihat seluruh order.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[92px]">Ref</TableHead>
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
                const basi = isStale(o, DEMO_NOW)
                const terlambat = hasLateRejected(o)
                const perluTindakan = basi || terlambat || o.cancellationRequested

                return (
                  <TableRow key={o.id} className="h-10">
                    <TableCell className="relative align-middle">
                      {/* Penanda urgensi hidup di kolom paling kiri, bukan di
                          warna status — dua sistem yang berbeda. */}
                      {perluTindakan ? (
                        <span className="absolute inset-y-1 left-0 w-0.5 rounded-full bg-attention" />
                      ) : null}
                      <Link
                        href={`/ops/order/${o.id}`}
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
                      {o.inspectorName ?? (
                        <span className="text-muted-foreground">Belum ada</span>
                      )}
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
                      {sejak(o.updatedAt, DEMO_NOW)}
                    </TableCell>

                    <TableCell className="align-middle text-right">
                      {o.status === "requested" ? (
                        <AssignDialog
                          orderRef={o.ref}
                          city={o.city}
                          inspectors={inspectors}
                          compact
                        />
                      ) : o.cancellationRequested ? (
                        <Link
                          href={`/ops/order/${o.id}`}
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
