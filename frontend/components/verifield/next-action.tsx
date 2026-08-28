"use client"

import * as React from "react"
import { CheckIcon, CloudOffIcon, LoaderIcon, TriangleAlertIcon } from "lucide-react"

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
import { cn } from "@/lib/utils"
import { useConnection } from "@/hooks/use-connection"
import { inspectorActions, STATUS_LABEL } from "@/lib/domain/status"
import type { Status } from "@/lib/domain/types"

type Kirim = { label: string; pending: boolean } | null

/** Inspektor tidak memilih status dari daftar. Ia menekan satu tombol yang sudah
    benar secara konteks — itu menghapus seluruh kelas kesalahan salah pilih,
    yang kalau terjadi harus dikoreksi koordinator secara manual (F-04). */
export function NextAction({ status }: { status: Status }) {
  const koneksi = useConnection()
  const [terkirim, setTerkirim] = React.useState<Kirim>(null)
  const actions = inspectorActions(status)
  const utama = actions.find((a) => a.tone === "primary")
  const kendala = actions.find((a) => a.to === "failed")

  if (terkirim) {
    return (
      <div
        className={cn(
          "flex flex-col items-center gap-2 rounded-2xl border p-6 text-center",
          terkirim.pending
            ? "border-attention/30 bg-attention/8"
            : "border-status-completed/30 bg-status-completed/8",
        )}
      >
        <span
          className={cn(
            "flex size-10 items-center justify-center rounded-full",
            terkirim.pending
              ? "bg-attention/15 text-attention"
              : "bg-status-completed/15 text-status-completed",
          )}
        >
          {terkirim.pending ? <CloudOffIcon className="size-5" /> : <CheckIcon className="size-5" />}
        </span>

        <span className="text-base font-semibold">{terkirim.label} tercatat</span>

        {terkirim.pending ? (
          <p className="text-xs leading-relaxed text-muted-foreground">
            Tersimpan di perangkat, menunggu terkirim. Waktu kejadian yang dipakai adalah saat
            Anda menekan tombol tadi, bukan saat sinyal kembali. Anda boleh lanjut bekerja.
          </p>
        ) : (
          <p className="text-xs leading-relaxed text-muted-foreground">
            Sudah diterima sistem. Klien melihat perubahan ini tanpa memuat ulang halaman.
          </p>
        )}
      </div>
    )
  }

  if (!utama) {
    return (
      <div className="rounded-2xl border border-border bg-muted/40 p-6 text-center">
        <p className="text-sm text-muted-foreground">
          Tidak ada tindakan tersisa. Order ini {STATUS_LABEL[status].toLowerCase()}.
        </p>
      </div>
    )
  }

  const offline = koneksi === "offline"

  return (
    <div className="flex flex-col gap-3">
      {offline ? (
        <p className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
          <CloudOffIcon className="size-3.5" />
          Sinyal tidak ada — laporan tetap tersimpan
        </p>
      ) : null}

      <Button
        size="xl"
        className="w-full"
        onClick={() => setTerkirim({ label: utama.label, pending: offline })}
      >
        {koneksi === "reconnecting" ? <LoaderIcon className="animate-spin" /> : null}
        {utama.label.toUpperCase()}
      </Button>

      {/* Melaporkan kendala menutup pekerjaan, jadi sengaja dibuat kecil dan
          butuh langkah tambahan. Tidak ada tombol batal di mana pun (B-04). */}
      {kendala ? (
        <Dialog>
          <DialogTrigger
            render={<Button variant="ghost" size="sm" className="mx-auto w-fit" />}
          >
            <TriangleAlertIcon data-icon="inline-start" />
            {kendala.label}
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
              placeholder="Apa yang menghalangi pekerjaan?"
              className="text-sm"
            />

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Kembali</DialogClose>
              <Button
                variant="destructive"
                onClick={() => setTerkirim({ label: "Kendala", pending: offline })}
              >
                Kirim Laporan
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  )
}

/** Muncul hanya ketika ada yang benar-benar mengantre. Kepastian bahwa laporan
    tidak hilang adalah kebutuhan utama inspektor. */
export function SyncQueueBanner({ count }: { count: number }) {
  const koneksi = useConnection()
  if (count === 0) return null

  return (
    <div className="mb-4 flex items-center gap-2.5 rounded-xl border border-attention/30 bg-attention/8 px-3 py-2.5">
      <CloudOffIcon className="size-4 shrink-0 text-attention" />
      <span className="text-xs leading-tight">
        <strong className="font-medium">{count} pembaruan menunggu terkirim.</strong>{" "}
        {koneksi === "offline"
          ? "Akan dikirim otomatis saat sinyal kembali."
          : "Sedang dikirim."}
      </span>
    </div>
  )
}
