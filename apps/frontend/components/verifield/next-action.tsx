"use client"

import * as React from "react"
import { CheckIcon, CloudOffIcon, TriangleAlertIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { useActor } from "@/components/verifield/actor-provider"
import { cn } from "@/lib/utils"
import { useOutbox } from "@/lib/offline/hooks"
import { enqueue } from "@/lib/offline/queue"
import { inspectorActions, STATUS_LABEL } from "@/lib/domain/status"
import type { JobOrder, Status } from "@/lib/domain/types"

/** Inspektor tidak memilih status dari daftar. Ia menekan satu tombol yang sudah
    benar secara konteks — itu menghapus seluruh kelas kesalahan salah pilih,
    yang kalau terjadi harus dikoreksi koordinator secara manual (F-04).

    Tombol selalu menerima ketukan, bahkan tanpa sinyal. Laporan masuk antrean di
    perangkat dengan waktu saat ditekan, lalu terkirim sendiri saat sambungan
    kembali. */
export function NextAction({ order }: { order: JobOrder }) {
  const actor = useActor()
  const outbox = useOutbox()

  const menunggu = outbox.items.filter((i) => i.orderId === order.id)
  const actions = inspectorActions(order.status)
  const utama = actions.find((a) => a.tone === "primary")
  const kendala = actions.find((a) => a.to === "failed")

  function laporkan(to: Status, reason?: string) {
    enqueue({
      clientEventId: crypto.randomUUID(),
      orderId: order.id,
      orderRef: order.ref,
      toStatus: to,
      // Waktu tombol ditekan, bukan waktu terkirim. Di area tanpa sinyal jarak
      // keduanya bisa berjam-jam, dan yang sah untuk laporan adalah yang ini.
      occurredAt: new Date().toISOString(),
      reason,
      actorId: actor.id,
    })
  }

  if (menunggu.length > 0) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-2xl border border-attention/30 bg-attention/8 p-6 text-center">
        <span className="flex size-10 items-center justify-center rounded-full bg-attention/15 text-attention">
          <CloudOffIcon className="size-5" />
        </span>
        <span className="text-base font-semibold">
          {menunggu.length === 1
            ? `${STATUS_LABEL[menunggu[0]!.toStatus]} tercatat`
            : `${menunggu.length} laporan tercatat`}
        </span>
        <p className="text-xs leading-relaxed text-muted-foreground">
          Tersimpan di perangkat, menunggu terkirim. Waktu kejadian yang dipakai adalah saat
          Anda menekan tombol tadi, bukan saat sinyal kembali. Anda boleh lanjut bekerja.
        </p>
      </div>
    )
  }

  if (!utama) {
    return (
      <div className="rounded-2xl border border-status-completed/30 bg-status-completed/8 p-6 text-center">
        <span className="mx-auto mb-2 flex size-10 items-center justify-center rounded-full bg-status-completed/15 text-status-completed">
          <CheckIcon className="size-5" />
        </span>
        <p className="text-sm text-muted-foreground">
          Tidak ada tindakan tersisa. Order ini {STATUS_LABEL[order.status].toLowerCase()}.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <Button size="xl" className="w-full" onClick={() => laporkan(utama.to)}>
        {utama.label.toUpperCase()}
      </Button>

      {/* Melaporkan kendala menutup pekerjaan, jadi sengaja dibuat kecil dan
          butuh langkah tambahan. Tidak ada tombol batal di mana pun (B-04). */}
      {kendala ? <KendalaDialog label={kendala.label} onSubmit={laporkan} /> : null}
    </div>
  )
}

function KendalaDialog({
  label,
  onSubmit,
}: {
  label: string
  onSubmit: (to: Status, reason: string) => void
}) {
  const [open, setOpen] = React.useState(false)
  const [reason, setReason] = React.useState("")

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="ghost" size="sm" className="mx-auto w-fit" />}>
        <TriangleAlertIcon data-icon="inline-start" />
        {label}
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Laporkan kendala</DialogTitle>
          <DialogDescription>
            Dipakai bila Anda sudah di lokasi tetapi pekerjaan tidak dapat dilaksanakan —
            kargo belum tiba, akses ditolak, cuaca membahayakan, atau objek tidak ditemukan.
          </DialogDescription>
        </DialogHeader>

        <Textarea
          rows={3}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Apa yang menghalangi pekerjaan?"
          className="text-sm"
        />

        <DialogFooter>
          <DialogClose render={<Button variant="secondary" />}>Kembali</DialogClose>
          <Button
            variant="destructive"
            disabled={reason.trim().length < 3}
            onClick={() => {
              onSubmit("failed", reason.trim())
              setOpen(false)
              setReason("")
            }}
          >
            Kirim Laporan
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Kepastian bahwa laporan tidak hilang adalah kebutuhan utama inspektor, jadi
    antrean ditampilkan apa adanya — termasuk ketika kosong sesudah semuanya
    terkirim, dan termasuk ketika ada yang ditolak. */
export function SyncQueueBanner() {
  const outbox = useOutbox()
  const { items, failures } = outbox

  if (items.length === 0 && failures.length === 0) return null

  return (
    <div className="mb-4 flex flex-col gap-2">
      {items.length > 0 ? (
        <div
          className={cn(
            "flex items-center gap-2.5 rounded-xl border px-3 py-2.5",
            "border-attention/30 bg-attention/8",
          )}
        >
          <CloudOffIcon className="size-4 shrink-0 text-attention" />
          <span className="text-xs leading-tight">
            <strong className="font-medium">
              {items.length} pembaruan menunggu terkirim.
            </strong>{" "}
            {outbox.flushing ? "Sedang dikirim." : "Akan dikirim otomatis saat sinyal kembali."}
          </span>
        </div>
      ) : null}

      {failures.map((f, i) => (
        <div
          key={`${f.orderRef}-${i}`}
          className="flex items-start gap-2.5 rounded-xl border border-border bg-muted/50 px-3 py-2.5"
        >
          <TriangleAlertIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          <span className="text-xs leading-relaxed">
            <strong className="font-medium">{f.orderRef}</strong> — {f.message}
          </span>
        </div>
      ))}
    </div>
  )
}
