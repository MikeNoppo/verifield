"use client"

import Link from "next/link"
import { ArrowLeftIcon, CloudOffIcon } from "lucide-react"

import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusStepper } from "@/components/verifield/status-rail"
import { EventTimeline } from "@/components/verifield/event-timeline"
import { AssignDialog } from "@/components/verifield/assign-dialog"
import { CancelDialog } from "@/components/verifield/cancel-dialog"
import { CancellationReview, CorrectionDialog } from "@/components/verifield/ops-actions"
import { useActor } from "@/components/verifield/actor-provider"
import { Card } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { useActorHref } from "@/lib/actor/hooks"
import { useLiveEvents, useLiveOrder } from "@/lib/live/hooks"
import { cancelAuthority, isStale, needsAssignment } from "@/lib/domain/status"
import type { Inspector, JobOrder, OpenAlertType } from "@/lib/domain/types"
import { sejak, tanggalLengkap } from "@/lib/format"

/** Kalimat yang harus dibaca koordinator berbeda per jenis alert, dan tindak
    lanjutnya juga berbeda: yang satu menyangkut kompensasi inspektor, yang lain
    menyangkut penyelesaian komersial dengan klien. */
const ALERT: Record<
  OpenAlertType,
  { Icon: typeof CloudOffIcon; judul: string; penjelasan: string }
> = {
  late_update_rejected: {
    Icon: CloudOffIcon,
    judul: "Pembaruan terlambat ditolak",
    penjelasan:
      "Order sudah final ketika laporan inspektor masuk, sehingga status tidak berubah. " +
      "Tetapi ada pekerjaan nyata yang telah dilakukan seseorang — kompensasi inspektor " +
      "perlu diselesaikan, dan situasinya dijelaskan kepada klien.",
  },
}

function Baris({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <dt className="text-[11px] text-muted-foreground">{label}</dt>
      <dd className="text-sm">{children}</dd>
    </div>
  )
}

export function OpsOrderDetail({
  order: initial,
  inspectors,
}: {
  order: JobOrder
  inspectors: Inspector[]
}) {
  const actor = useActor()
  const href = useActorHref()
  const order = useLiveOrder(initial)
  const events = useLiveEvents(order, actor.id)

  const now = new Date()
  const auth = cancelAuthority("admin", order.status)
  const basi = isStale(order, now)
  const alert = order.openAlertType ? ALERT[order.openAlertType] : null

  return (
    <>
      <Link
        href={href("/ops")}
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
            {order.clientName} · {order.inspectionType} · {order.location}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {needsAssignment(order) ? (
            <AssignDialog order={order} inspectors={inspectors} />
          ) : null}
          <CorrectionDialog order={order} />
          {auth.kind === "allowed" ? <CancelDialog role="admin" order={order} /> : null}
        </div>
      </div>

      {order.pendingCancellation ? (
        <div className="mb-4">
          <CancellationReview order={order} request={order.pendingCancellation} />
        </div>
      ) : null}

      {alert ? (
        <div className="mb-4 flex gap-3 rounded-xl border border-attention/30 bg-attention/6 p-4">
          <alert.Icon className="mt-0.5 size-4 shrink-0 text-attention" />
          <div className="flex flex-col gap-1">
            <span className="text-sm font-medium">{alert.judul}</span>
            <p className="text-xs leading-relaxed text-muted-foreground">{alert.penjelasan}</p>
          </div>
        </div>
      ) : null}

      {basi ? (
        <div className="mb-4 rounded-xl border border-attention/30 bg-attention/6 px-4 py-3 text-xs leading-relaxed">
          <strong className="font-medium">
            Tanpa pembaruan {sejak(order.statusChangedAt, now)}.
          </strong>{" "}
          Kemungkinan inspektor lupa memperbarui. Tindak lanjuti sebelum klien bertanya.
        </div>
      ) : null}

      <div className="grid gap-5 lg:grid-cols-[320px_1fr]">
        <div className="flex flex-col gap-5">
          <Card className="gap-0 p-4">
            <dl className="divide-y divide-border">
              <Baris label="Objek diperiksa">{order.object}</Baris>
              <Baris label="Lokasi">
                {order.location}, {order.city}
              </Baris>
              <Baris label="Alamat">
                <span className="text-xs text-muted-foreground">{order.address}</span>
              </Baris>
              <Baris label="Jadwal diminta">
                <span className="tabular">{tanggalLengkap(order.scheduledAt)}</span>
              </Baris>
              <Baris label="Inspektor ditugaskan">
                {order.inspectorName ?? (
                  <span className="text-muted-foreground">Belum ditugaskan</span>
                )}
              </Baris>
              <Baris label="Versi data">
                <span className="tabular font-mono text-xs text-muted-foreground">
                  v{order.version} · dipakai menolak perubahan bersamaan
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
              Jejak Audit
            </h2>
            <p className="text-[11px] leading-relaxed text-muted-foreground">
              Riwayat bersifat menambah — tidak pernah menimpa atau menghapus. Penanda unik
              perangkat dan waktu penerimaan ikut ditampilkan di sini, tetapi tidak kepada klien.
            </p>
          </div>
          <Separator />
          <EventTimeline events={events} showAudit />
        </Card>
      </div>
    </>
  )
}
