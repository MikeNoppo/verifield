"use client"

import Link from "next/link"
import { ArrowLeftIcon } from "lucide-react"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusStepper } from "@/components/verifield/status-rail"
import { EventTimeline } from "@/components/verifield/event-timeline"
import { CancelDialog } from "@/components/verifield/cancel-dialog"
import { useActor } from "@/components/verifield/actor-provider"
import { Card } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveEvents, useLiveOrder } from "@/lib/live/hooks"
import { cancelAuthority, cancelOffered } from "@/lib/domain/status"
import type { JobOrder } from "@/lib/domain/types"
import { tanggalLengkap } from "@/lib/format"

function Baris({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <dt className="text-[11px] text-muted-foreground">{label}</dt>
      <dd className="text-sm">{children}</dd>
    </div>
  )
}

export function KlienOrderDetail({ order: initial }: { order: JobOrder }) {
  const actor = useActor()
  const href = useActorHref()
  const order = useLiveOrder(initial)
  const events = useLiveEvents(order, actor.id)
  const auth = cancelAuthority("client", order.status)

  return (
    <>
      <Link
        href={href("/klien")}
        className="mb-4 inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
      >
        <ArrowLeftIcon className="size-3.5" />
        Kembali ke daftar order
      </Link>

      <div className="mb-5 flex flex-wrap items-start justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2.5">
            <h1 className="tabular font-mono text-lg font-semibold tracking-tight">
              {order.ref}
            </h1>
            <StatusBadge status={order.status} />
          </div>
          <p className="text-sm text-muted-foreground">
            {order.inspectionType} · {order.location}
          </p>
        </div>

        <div className="flex items-center gap-2">
          {order.cancellationRequested ? (
            <span className="rounded-4xl border border-attention/30 bg-attention/10 px-3 py-1 text-xs font-medium text-attention">
              Permintaan pembatalan sedang ditinjau
            </span>
          ) : cancelOffered(auth) ? (
            <CancelDialog role="client" order={order} />
          ) : null}
        </div>
      </div>

      <div className="grid gap-5 lg:grid-cols-[320px_1fr]">
        <div className="flex flex-col gap-5">
          <Card className="gap-0 p-4">
            <dl className="divide-y divide-border">
              <Baris label="Objek diperiksa">{order.object}</Baris>
              <Baris label="Lokasi">
                {order.location}, {order.city}
              </Baris>
              <Baris label="Jadwal diminta">
                <span className="tabular">{tanggalLengkap(order.scheduledAt)}</span>
              </Baris>
              {/* Identitas lengkap dan posisi inspektor tidak ditampilkan kepada
                  klien — kebutuhannya sudah terpenuhi oleh status (B-08). */}
              <Baris label="Inspektor">
                <span className="text-muted-foreground">
                  {order.inspectorName ? "Sudah ditugaskan" : "Belum ditugaskan"}
                </span>
              </Baris>
            </dl>
          </Card>

          <Card className="gap-3 p-4">
            <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Tahapan
            </h2>
            <Separator />
            <StatusStepper order={order} />
          </Card>
        </div>

        <Card className="gap-3 p-4">
          <div className="flex flex-col gap-1">
            <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Riwayat Status
            </h2>
            <p className="text-[11px] leading-relaxed text-muted-foreground">
              Waktu yang ditampilkan besar adalah waktu kejadian di lapangan. Bila pembaruan
              terkirim terlambat karena sinyal, waktu penerimaannya ikut dicatat di bawahnya.
            </p>
          </div>
          <Separator />
          <EventTimeline events={events} />
        </Card>
      </div>
    </>
  )
}
