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
import { cancelAuthority } from "@/lib/domain/status"
import type { Role, Status } from "@/lib/domain/types"

/** Kewenangan dibaca dari matriks di lib/domain/status.ts, tidak ditebak di sini.
    Tiga hasilnya berbeda secara mendasar: boleh langsung, harus ditinjau
    koordinator (B-05), atau tidak boleh sama sekali — dan masing-masing punya
    kalimatnya sendiri yang bisa dimengerti orang non-teknis (F-05). */
export function CancelDialog({
  role,
  status,
  orderRef,
}: {
  role: Role
  status: Status
  orderRef: string
}) {
  const [open, setOpen] = React.useState(false)
  const [terkirim, setTerkirim] = React.useState(false)
  const auth = cancelAuthority(role, status)

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) setTerkirim(false)
  }

  return (
    <Dialog open={open} onOpenChange={tutup}>
      <DialogTrigger
        render={
          <Button variant="destructive" size="sm" disabled={auth.kind === "forbidden"} />
        }
      >
        {auth.kind === "requires_approval" ? "Ajukan Pembatalan" : "Batalkan Order"}
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {terkirim ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                {auth.kind === "requires_approval"
                  ? "Permintaan diteruskan"
                  : "Order dibatalkan"}
              </DialogTitle>
              <DialogDescription>
                {auth.kind === "requires_approval"
                  ? `Permintaan pembatalan ${orderRef} sudah diteruskan ke koordinator. Anda akan melihat keputusannya pada riwayat order ini.`
                  : `${orderRef} telah dibatalkan dan tercatat pada riwayat order.`}
              </DialogDescription>
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
                  : `Batalkan ${orderRef}?`}
              </DialogTitle>
              <DialogDescription>
                {auth.kind === "allowed" ? auth.consequence : null}
                {auth.kind === "requires_approval" ? auth.message : null}
                {auth.kind === "forbidden" ? auth.message : null}
              </DialogDescription>
            </DialogHeader>

            {auth.kind === "allowed" && auth.fee !== "none" ? (
              <p className="flex items-start gap-2 rounded-lg border border-attention/25 bg-attention/8 px-3 py-2 text-xs leading-relaxed">
                <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-attention" />
                Biaya ini muncul karena inspektor sudah dikerahkan. Membatalkan sekarang tetap
                menutup order.
              </p>
            ) : null}

            {auth.kind === "requires_approval" ? (
              <div className="flex flex-col gap-2">
                <label htmlFor="alasan" className="text-xs font-medium">
                  Alasan pembatalan
                </label>
                <Textarea
                  id="alasan"
                  rows={3}
                  placeholder="Contoh: jadwal pengapalan dimajukan."
                  className="text-sm"
                />
                <p className="flex items-start gap-2 text-[11px] leading-relaxed text-muted-foreground">
                  <InfoIcon className="mt-0.5 size-3 shrink-0" />
                  Pekerjaan tetap berjalan sampai koordinator memutuskan.
                </p>
              </div>
            ) : null}

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Kembali</DialogClose>
              <Button
                variant="destructive"
                onClick={() => setTerkirim(true)}
                disabled={auth.kind === "forbidden"}
              >
                {auth.kind === "requires_approval" ? "Kirim Permintaan" : "Ya, Batalkan"}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
