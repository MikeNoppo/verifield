"use client"

import * as React from "react"
import { CircleCheckIcon, InfoIcon, TriangleAlertIcon } from "lucide-react"

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
import { Spinner } from "@/components/ui/spinner"
import { useActor } from "@/components/verifield/actor-provider"
import { ApiError } from "@/lib/api/client"
import { cancelOrder, type CancelResult } from "@/lib/api/orders"
import { useApplyResult } from "@/lib/live/hooks"
import { cancelAuthority } from "@/lib/domain/status"
import type { JobOrder, Role } from "@/lib/domain/types"

/** Kewenangan dibaca dari matriks di lib/domain/status.ts, tidak ditebak di sini.
    Tiga hasilnya berbeda secara mendasar: boleh langsung, harus ditinjau
    koordinator (B-05), atau tidak boleh sama sekali — dan masing-masing punya
    kalimatnya sendiri yang bisa dimengerti orang non-teknis (F-05).

    Matriks yang sama ditegakkan ulang di server, karena tampilan yang benar tidak
    pernah cukup menjadi jaminan. */
export function CancelDialog({ role, order }: { role: Role; order: JobOrder }) {
  const actor = useActor()
  const terapkan = useApplyResult()

  const [open, setOpen] = React.useState(false)
  const [alasan, setAlasan] = React.useState("")
  const [kirim, setKirim] = React.useState(false)
  const [hasil, setHasil] = React.useState<CancelResult | null>(null)
  const [galat, setGalat] = React.useState<string | null>(null)

  const auth = cancelAuthority(role, order.status)

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) {
      setHasil(null)
      setGalat(null)
      setAlasan("")
    }
  }

  async function kirimkan() {
    setKirim(true)
    setGalat(null)
    try {
      const result = await cancelOrder(actor.id, order.id, alasan.trim(), order.version)
      terapkan(result.order)
      setHasil(result)
    } catch (error) {
      setGalat(
        error instanceof ApiError
          ? error.message
          : "Permintaan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      )
    } finally {
      setKirim(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={tutup}>
      <DialogTrigger
        render={<Button variant="destructive" size="sm" disabled={auth.kind === "forbidden"} />}
      >
        {auth.kind === "requires_approval" ? "Ajukan Pembatalan" : "Batalkan Order"}
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {hasil ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                {hasil.status === "pending_approval"
                  ? "Permintaan diteruskan"
                  : "Order dibatalkan"}
              </DialogTitle>
              {/* Kalimatnya berasal dari server, bukan disusun ulang di sini —
                  satu tempat yang menentukan bagaimana penolakan dijelaskan. */}
              <DialogDescription>{hasil.message}</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>
                {auth.kind === "requires_approval"
                  ? "Ajukan pembatalan"
                  : `Batalkan ${order.ref}?`}
              </DialogTitle>
              <DialogDescription>
                {auth.kind === "allowed" ? auth.consequence : auth.message}
              </DialogDescription>
            </DialogHeader>

            {auth.kind === "allowed" && auth.fee !== "none" ? (
              <p className="flex items-start gap-2 rounded-lg border border-attention/25 bg-attention/8 px-3 py-2 text-xs leading-relaxed">
                <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-attention" />
                Biaya ini muncul karena inspektor sudah dikerahkan. Membatalkan sekarang tetap
                menutup order.
              </p>
            ) : null}

            <div className="flex flex-col gap-2">
              <label htmlFor="alasan-batal" className="text-xs font-medium">
                Alasan pembatalan <span className="text-destructive">*</span>
              </label>
              <Textarea
                id="alasan-batal"
                rows={3}
                value={alasan}
                onChange={(e) => setAlasan(e.target.value)}
                placeholder="Contoh: jadwal pengapalan dimajukan."
                className="text-sm"
              />
              {auth.kind === "requires_approval" ? (
                <p className="flex items-start gap-2 text-[11px] leading-relaxed text-muted-foreground">
                  <InfoIcon className="mt-0.5 size-3 shrink-0" />
                  Pekerjaan tetap berjalan sampai koordinator memutuskan.
                </p>
              ) : null}
            </div>

            {galat ? (
              <p className="rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
                {galat}
              </p>
            ) : null}

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Kembali</DialogClose>
              <Button
                variant="destructive"
                onClick={kirimkan}
                disabled={kirim || alasan.trim().length < 3 || auth.kind === "forbidden"}
              >
                {kirim ? <Spinner /> : null}
                {auth.kind === "requires_approval" ? "Kirim Permintaan" : "Ya, Batalkan"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
