"use client"

import Link from "next/link"
import { ArrowLeftIcon, MapPinIcon, TriangleAlertIcon } from "lucide-react"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusStepper } from "@/components/verifield/status-rail"
import { NextAction, SyncQueueBanner } from "@/components/verifield/next-action"
import { Card } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveOrder } from "@/lib/live/hooks"
import { isTerminal } from "@/lib/domain/status"
import type { JobOrder } from "@/lib/domain/types"
import { formatFullDateTime } from "@/lib/format"

export function LapanganOrderDetail({ order: initial }: { order: JobOrder }) {
  const order = useLiveOrder(initial)
  const href = useActorHref()

  return (
    <>
      <Link
        href={href("/lapangan")}
        className="mb-4 inline-flex min-h-8 items-center gap-1.5 text-xs text-muted-foreground"
      >
        <ArrowLeftIcon className="size-3.5" />
        Tugas saya
      </Link>

      <SyncQueueBanner />

      {/* B-05 memutuskan pekerjaan tetap berjalan selama permintaan pembatalan
          ditinjau — permintaannya bisa saja ditolak. Tetapi berjalan terus
          tanpa memberi tahu inspektor membuat tabrakan "selesai mendahului
          keputusan" jadi kejadian rutin, bukan langka. Memberi tahu tanpa
          memblokir sudah cukup menurunkannya (B-10). */}
      {order.cancellationRequested && !isTerminal(order.status) ? (
        <div className="mb-4 flex gap-3 rounded-xl border border-attention/30 bg-attention/6 p-4">
          <TriangleAlertIcon className="mt-0.5 size-4 shrink-0 text-attention" />
          <div className="flex flex-col gap-1">
            <span className="text-sm font-medium">Klien mengajukan pembatalan</span>
            <p className="text-xs leading-relaxed text-muted-foreground">
              Koordinator sedang meninjau. Hubungi koordinator sebelum melanjutkan — bila
              Anda menyelesaikan pekerjaan lebih dulu, statusnya tidak dapat diubah lagi.
            </p>
          </div>
        </div>
      ) : null}

      <div className="mb-5 flex flex-col gap-2">
        <div className="flex items-center justify-between gap-3">
          <span className="tabular font-mono text-xs text-muted-foreground">{order.ref}</span>
          <StatusBadge status={order.status} size="sm" />
        </div>
        <h1 className="text-lg leading-snug font-semibold tracking-tight">{order.object}</h1>
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <MapPinIcon className="size-3.5 shrink-0" />
          {order.location}, {order.city}
        </span>
        <span className="tabular text-xs text-muted-foreground">
          Dijadwalkan {formatFullDateTime(order.scheduledAt)}
        </span>
      </div>

      {/* Satu tombol besar berisi tindakan next. Tidak ada daftar status
          untuk dipilih, dan tidak ada tombol batal (B-04, F-04). */}
      <div className="mb-6">
        <NextAction order={order} />
      </div>

      <Card className="gap-3 p-4">
        <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Tahapan
        </h2>
        <Separator />
        <StatusStepper order={order} />
      </Card>
    </>
  )
}
