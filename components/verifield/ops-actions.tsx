"use client"

import * as React from "react"
import { CircleCheckIcon, PencilLineIcon } from "lucide-react"

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
import { PIPELINE, STATUS_LABEL } from "@/lib/domain/status"
import type { Status } from "@/lib/domain/types"

/** Koreksi wajib beralasan dan tercatat sebagai entri baru, bukan menghapus
    entri lama. Tanpa jalur resmi ini koordinator akan mengubah data langsung
    ke basis data dan seluruh nilai jejak audit hilang (B-06, F-06). */
export function CorrectionDialog({
  orderRef,
  status,
}: {
  orderRef: string
  status: Status
}) {
  const [open, setOpen] = React.useState(false)
  const [selesai, setSelesai] = React.useState(false)
  const [tujuan, setTujuan] = React.useState<string>(status)
  const [alasan, setAlasan] = React.useState("")

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) {
      setSelesai(false)
      setAlasan("")
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
                {orderRef} dikembalikan ke {STATUS_LABEL[tujuan as Status]}. Entri lama tetap
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
              <DialogTitle>Koreksi status {orderRef}</DialogTitle>
              <DialogDescription>
                Kesalahan input di lapangan pasti terjadi. Koreksi tidak menghapus apa pun — ia
                ditambahkan sebagai entri baru pada riwayat.
              </DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-medium">Kembalikan ke</label>
                <Select value={tujuan} onValueChange={(v) => setTujuan(String(v))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PIPELINE.map((s) => (
                      <SelectItem key={s} value={s}>
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
            </div>

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Batal</DialogClose>
              <Button disabled={alasan.trim().length < 8} onClick={() => setSelesai(true)}>
                Simpan Koreksi
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

/** Permintaan pembatalan yang masuk saat pekerjaan sudah berjalan. Koordinator
    memutuskan setelah aspek komersialnya jelas (B-05). */
export function CancellationReview({
  orderRef,
  reason,
}: {
  orderRef: string
  reason?: string
}) {
  const [keputusan, setKeputusan] = React.useState<"pending" | "setuju" | "tolak">("pending")

  if (keputusan !== "pending") {
    return (
      <div className="flex flex-col gap-1.5 rounded-xl border border-border bg-muted/40 p-4">
        <span className="flex items-center gap-2 text-sm font-medium">
          <CircleCheckIcon className="size-4 text-status-completed" />
          {keputusan === "setuju" ? "Pembatalan disetujui" : "Permintaan ditolak"}
        </span>
        <p className="text-xs leading-relaxed text-muted-foreground">
          {keputusan === "setuju"
            ? `${orderRef} ditutup sebagai Dibatalkan. Klien melihat keputusan ini pada riwayat order.`
            : `${orderRef} berjalan kembali sebagai Sedang Dikerjakan. Alasan penolakan tercatat pada riwayat.`}
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-attention/30 bg-attention/6 p-4">
      <div className="flex flex-col gap-1">
        <span className="text-sm font-medium">Permintaan pembatalan menunggu keputusan</span>
        <p className="text-xs leading-relaxed text-muted-foreground">
          Pekerjaan sudah dimulai, sehingga sudah ada biaya nyata: waktu inspektor, perjalanan,
          dan kemungkinan sampel yang telah diambil.
        </p>
      </div>

      {reason ? (
        <p className="rounded-lg border border-border bg-background px-3 py-2 text-xs leading-relaxed">
          {reason}
        </p>
      ) : null}

      <div className="flex gap-2">
        <Button size="sm" variant="destructive" onClick={() => setKeputusan("setuju")}>
          Setujui Pembatalan
        </Button>
        <Button size="sm" variant="outline" onClick={() => setKeputusan("tolak")}>
          Tolak, Lanjutkan Pekerjaan
        </Button>
      </div>
    </div>
  )
}
