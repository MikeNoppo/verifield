"use client"

import * as React from "react"
import { CircleCheckIcon, HandshakeIcon, PencilLineIcon, TriangleAlertIcon } from "lucide-react"

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { Spinner } from "@/components/ui/spinner"
import { useActor } from "@/components/verifield/actor-provider"
import { ApiError } from "@/lib/api/client"
import { correctStatus, decideCancellation, settleCancellation } from "@/lib/api/orders"
import { useApplyResult } from "@/lib/live/hooks"
import { PIPELINE, STATUS_LABEL } from "@/lib/domain/status"
import type {
  JobOrder,
  PendingCancellation,
  SettlementOutcome,
  Status,
} from "@/lib/domain/types"
import { tanggalLengkap } from "@/lib/format"
import { cn } from "@/lib/utils"

/** Koreksi wajib beralasan dan tercatat sebagai entri baru, bukan menghapus
    entri lama. Tanpa jalur resmi ini koordinator akan mengubah data langsung
    ke basis data dan seluruh nilai jejak audit hilang (B-06, F-06). */
export function CorrectionDialog({ order }: { order: JobOrder }) {
  const actor = useActor()
  const terapkan = useApplyResult()

  const [open, setOpen] = React.useState(false)
  const [tujuan, setTujuan] = React.useState<string>(order.status)
  const [alasan, setAlasan] = React.useState("")
  const [kirim, setKirim] = React.useState(false)
  const [selesai, setSelesai] = React.useState(false)
  const [galat, setGalat] = React.useState<string | null>(null)

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) {
      setSelesai(false)
      setGalat(null)
      setAlasan("")
    }
  }

  async function simpan() {
    setKirim(true)
    setGalat(null)
    try {
      const terbaru = await correctStatus(
        actor.id,
        order.id,
        tujuan as Status,
        alasan.trim(),
        order.version,
      )
      terapkan(terbaru)
      setSelesai(true)
    } catch (error) {
      setGalat(
        error instanceof ApiError
          ? error.message
          : "Koreksi tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      )
    } finally {
      setKirim(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={tutup}>
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        <PencilLineIcon data-icon="inline-start" />
        Koreksi Status
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {selesai ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                Koreksi tercatat
              </DialogTitle>
              <DialogDescription>
                {order.ref} dikembalikan ke {STATUS_LABEL[tujuan as Status]}. Entri lama tetap
                utuh pada riwayat, dan koreksi ini muncul sebagai entri baru beserta alasannya.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Koreksi status {order.ref}</DialogTitle>
              <DialogDescription>
                Kesalahan input di lapangan pasti terjadi. Koreksi tidak menghapus apa pun — ia
                ditambahkan sebagai entri baru pada riwayat.
              </DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-medium">Kembalikan ke</label>
                <Select value={tujuan} onValueChange={(v) => setTujuan(String(v))}>
                  <SelectTrigger className="w-full">
                    <SelectValue>{STATUS_LABEL[tujuan as Status]}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {PIPELINE.filter((s) => s !== order.status).map((s) => (
                      <SelectItem key={s} value={s} label={STATUS_LABEL[s]}>
                        {STATUS_LABEL[s]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex flex-col gap-1.5">
                <label htmlFor="alasan-koreksi" className="text-xs font-medium">
                  Alasan koreksi <span className="text-destructive">*</span>
                </label>
                <Textarea
                  id="alasan-koreksi"
                  rows={3}
                  value={alasan}
                  onChange={(e) => setAlasan(e.target.value)}
                  placeholder="Contoh: inspektor menghubungi koordinator, tombol Selesai tertekan tidak sengaja."
                  className="text-sm"
                />
              </div>

              {galat ? (
                <p className="rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
                  {galat}
                </p>
              ) : null}
            </div>

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Batal</DialogClose>
              <Button
                disabled={kirim || alasan.trim().length < 8 || tujuan === order.status}
                onClick={simpan}
              >
                {kirim ? <Spinner /> : null}
                Simpan Koreksi
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

const HASIL: { value: SettlementOutcome; label: string; jelas: string }[] = [
  {
    value: "billed_full",
    label: "Tagih penuh",
    jelas: "Pekerjaan terkirim utuh sebelum pembatalan sempat diproses.",
  },
  {
    value: "billed_partial",
    label: "Tagih sebagian",
    jelas: "Kompromi komersial — sebagian biaya ditanggung klien.",
  },
  {
    value: "waived",
    label: "Bebaskan biaya",
    jelas: "Permintaan klien masuk cukup awal; biaya tidak ditagihkan.",
  },
]

/** Pekerjaan terlanjur selesai mendahului keputusan pembatalan (B-10).
    Statusnya sudah final dan tidak diutak-atik di sini — yang dijawab adalah
    pertanyaan komersial yang tertinggal, dan jawabannya masuk riwayat supaya
    klien serta CS membaca hal yang sama. */
function SettlementReview({
  order,
  request,
}: {
  order: JobOrder
  request: PendingCancellation
}) {
  const actor = useActor()
  const terapkan = useApplyResult()

  const [hasil, setHasil] = React.useState<SettlementOutcome | null>(null)
  const [catatan, setCatatan] = React.useState("")
  const [kirim, setKirim] = React.useState(false)
  const [galat, setGalat] = React.useState<string | null>(null)

  async function simpan() {
    if (!hasil) return
    setKirim(true)
    setGalat(null)
    try {
      terapkan(await settleCancellation(actor.id, order.id, request.id, hasil, catatan.trim()))
    } catch (error) {
      setGalat(
        error instanceof ApiError
          ? error.message
          : "Keputusan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      )
    } finally {
      setKirim(false)
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-attention/30 bg-attention/6 p-4">
      <div className="flex flex-col gap-1">
        <span className="flex items-center gap-2 text-sm font-medium">
          <HandshakeIcon className="size-4 text-attention" />
          Pekerjaan selesai sebelum pembatalan diputuskan
        </span>
        <p className="text-xs leading-relaxed text-muted-foreground">
          Status {order.ref} tidak dapat diubah lagi — pekerjaannya memang dikerjakan. Yang
          belum terjawab adalah biayanya. Klien mengajukan pembatalan selagi pekerjaan masih
          berjalan, dan yang membuat pekerjaan tetap jalan adalah aturan kita sendiri.
        </p>
      </div>

      <div className="rounded-lg border border-border bg-background px-3 py-2">
        <p className="text-xs leading-relaxed">{request.reason}</p>
        <p className="tabular mt-1.5 text-[11px] text-muted-foreground">
          Diajukan {request.requestedByName} · {tanggalLengkap(request.createdAt)}
        </p>
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">Penyelesaian</span>
        <div className="flex flex-col gap-1.5">
          {HASIL.map((h) => (
            <button
              key={h.value}
              type="button"
              onClick={() => setHasil(h.value)}
              className={cn(
                "flex flex-col gap-0.5 rounded-lg border px-3 py-2 text-left transition-colors",
                hasil === h.value
                  ? "border-attention bg-attention/10"
                  : "border-border bg-background hover:bg-muted/50",
              )}
            >
              <span className="text-xs font-medium">{h.label}</span>
              <span className="text-[11px] leading-relaxed text-muted-foreground">{h.jelas}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="catatan-penyelesaian" className="text-xs font-medium">
          Catatan <span className="text-destructive">*</span>
        </label>
        <Textarea
          id="catatan-penyelesaian"
          rows={2}
          value={catatan}
          onChange={(e) => setCatatan(e.target.value)}
          placeholder="Contoh: disepakati lewat telepon dengan Ibu Dewi, ditagih separuh biaya kunjungan."
          className="text-sm"
        />
        <p className="text-[11px] leading-relaxed text-muted-foreground">
          Catatan ini yang dibaca klien dan CS bila kemudian dipersoalkan.
        </p>
      </div>

      {galat ? (
        <p className="rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
          {galat}
        </p>
      ) : null}

      <div>
        <Button size="sm" disabled={kirim || !hasil || catatan.trim().length < 8} onClick={simpan}>
          {kirim ? <Spinner /> : null}
          Catat Penyelesaian
        </Button>
      </div>
    </div>
  )
}

/** Permintaan pembatalan yang masuk saat pekerjaan sudah berjalan. Koordinator
    memutuskan setelah aspek komersialnya jelas (B-05). */
export function CancellationReview({
  order,
  request,
}: {
  order: JobOrder
  request: PendingCancellation
}) {
  if (request.status === "pending_settlement") {
    return <SettlementReview order={order} request={request} />
  }
  return <PembatalanReview order={order} request={request} />
}

function PembatalanReview({
  order,
  request,
}: {
  order: JobOrder
  request: PendingCancellation
}) {
  const actor = useActor()
  const terapkan = useApplyResult()

  const [kirim, setKirim] = React.useState<"approve" | "reject" | null>(null)
  const [galat, setGalat] = React.useState<string | null>(null)

  async function putuskan(decision: "approve" | "reject") {
    setKirim(decision)
    setGalat(null)
    try {
      terapkan(await decideCancellation(actor.id, order.id, request.id, decision))
      // Hasilnya tidak ditampilkan sebagai layar konfirmasi tersendiri: begitu
      // store diperbarui, panel ini hilang dan riwayat di sebelahnya bertambah
      // satu entri. Itu umpan balik yang lebih jujur daripada kalimat sukses.
    } catch (error) {
      setGalat(
        error instanceof ApiError
          ? error.message
          : "Keputusan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      )
    } finally {
      setKirim(null)
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-attention/30 bg-attention/6 p-4">
      <div className="flex flex-col gap-1">
        <span className="flex items-center gap-2 text-sm font-medium">
          <TriangleAlertIcon className="size-4 text-attention" />
          Permintaan pembatalan menunggu keputusan
        </span>
        <p className="text-xs leading-relaxed text-muted-foreground">
          Pekerjaan sudah dimulai, sehingga sudah ada biaya nyata: waktu inspektor, perjalanan,
          dan kemungkinan sampel yang telah diambil.
        </p>
      </div>

      <div className="rounded-lg border border-border bg-background px-3 py-2">
        <p className="text-xs leading-relaxed">{request.reason}</p>
        <p className="tabular mt-1.5 text-[11px] text-muted-foreground">
          {request.requestedByName} · {tanggalLengkap(request.createdAt)}
        </p>
      </div>

      {galat ? (
        <p className="rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
          {galat}
        </p>
      ) : null}

      <div className="flex gap-2">
        <Button
          size="sm"
          variant="destructive"
          disabled={kirim !== null}
          onClick={() => putuskan("approve")}
        >
          {kirim === "approve" ? <Spinner /> : null}
          Setujui Pembatalan
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={kirim !== null}
          onClick={() => putuskan("reject")}
        >
          {kirim === "reject" ? <Spinner /> : null}
          Tolak, Lanjutkan Pekerjaan
        </Button>
      </div>
    </div>
  )
}
