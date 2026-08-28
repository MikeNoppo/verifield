"use client"

import Link from "next/link"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusRail } from "@/components/verifield/status-rail"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveList } from "@/lib/live/hooks"
import { inBucket, type Bucket } from "@/lib/domain/summary"
import type { JobOrder } from "@/lib/domain/types"
import { relativeTime, formatDateTime } from "@/lib/format"

/** Penyaringan dilakukan di sini, bukan di server, supaya baris yang statusnya
    berubah lewat stream langsung berpindah saringan tanpa memuat ulang halaman.
    Itu justru inti fitur ini: status berubah tanpa klien melakukan apa pun. */
export function KlienOrderList({
  orders: initial,
  bucket,
}: {
  orders: JobOrder[]
  bucket?: Bucket
}) {
  const semua = useLiveList(initial)
  const orders = semua.filter((o) => inBucket(o, bucket))
  const href = useActorHref()
  const now = new Date()

  if (orders.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>Tidak ada order pada saringan ini</EmptyTitle>
          <EmptyDescription>
            Coba pilih saringan lain, atau buat permintaan inspeksi baru.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl border border-border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="w-[124px]">Ref</TableHead>
            <TableHead>Objek &amp; Lokasi</TableHead>
            <TableHead className="hidden w-[132px] lg:table-cell">Jadwal</TableHead>
            <TableHead className="w-[228px]">Status</TableHead>
            <TableHead className="hidden w-[116px] text-right sm:table-cell">
              Diperbarui
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {orders.map((o) => (
            <TableRow key={o.id} className="h-12">
              <TableCell className="align-middle">
                <Link
                  href={href(`/klien/order/${o.id}`)}
                  prefetch={false}
                  className="tabular font-mono text-xs font-medium hover:underline"
                >
                  {o.ref}
                </Link>
              </TableCell>

              <TableCell className="align-middle">
                <Link
                  href={href(`/klien/order/${o.id}`)}
                  prefetch={false}
                  className="flex flex-col gap-0.5"
                >
                  <span className="truncate text-sm">{o.object}</span>
                  <span className="truncate text-[11px] text-muted-foreground">
                    {o.inspectionType} · {o.location}
                  </span>
                </Link>
              </TableCell>

              <TableCell className="tabular hidden align-middle text-xs text-muted-foreground lg:table-cell">
                {formatDateTime(o.scheduledAt)}
              </TableCell>

              {/* Rel menjawab "orangnya sudah berangkat belum" tanpa perlu
                  dibaca; badge di sebelahnya yang membawa nama statusnya. */}
              <TableCell className="align-middle">
                <div className="flex items-center gap-3">
                  <StatusRail order={o} className="w-20 shrink-0" />
                  <StatusBadge status={o.status} size="sm" />
                </div>
              </TableCell>

              <TableCell className="tabular hidden text-right align-middle text-xs text-muted-foreground sm:table-cell">
                {relativeTime(o.statusChangedAt, now)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
