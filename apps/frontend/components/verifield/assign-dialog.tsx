"use client"

import * as React from "react"
import { CircleCheckIcon, TriangleAlertIcon, UserPlusIcon } from "lucide-react"

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
import { Spinner } from "@/components/ui/spinner"
import { useActor } from "@/components/verifield/actor-provider"
import { ApiError } from "@/lib/api/client"
import { assignInspector } from "@/lib/api/orders"
import { useApplyResult } from "@/lib/live/hooks"
import type { Inspector, JobOrder } from "@/lib/domain/types"

type Hasil = { kind: "ok"; nama: string } | { kind: "conflict"; pesan: string } | null

/** Penugasan membawa versi order yang sedang dilihat. Bila koordinator lain
    sudah menugaskan lebih dulu, server menolak dan penolakannya ditampilkan apa
    adanya — menerima keduanya berarti satu penugasan hilang tanpa ada yang
    menyadari, dan dua inspektor berangkat ke lokasi yang sama (B-09). */
export function AssignDialog({
  order,
  inspectors,
  compact = false,
}: {
  order: JobOrder
  inspectors: Inspector[]
  compact?: boolean
}) {
  const actor = useActor()
  const terapkan = useApplyResult()

  const [open, setOpen] = React.useState(false)
  const [kirim, setKirim] = React.useState(false)
  const [hasil, setHasil] = React.useState<Hasil>(null)

  // Beban kerja jadi dasar urutan karena penugasan otomatis berada di luar
  // cakupan — angka inilah yang menggantikan pertimbangan itu.
  const urut = React.useMemo(
    () => [...inspectors].sort((a, b) => a.activeJobs - b.activeJobs),
    [inspectors],
  )
  const [pilihan, setPilihan] = React.useState(urut[0]?.id ?? "")

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) setHasil(null)
  }

  async function tugaskan() {
    setKirim(true)
    try {
      const terbaru = await assignInspector(actor.id, order.id, pilihan, order.version)
      terapkan(terbaru)
      setHasil({ kind: "ok", nama: terbaru.inspectorName ?? "Inspektor" })
    } catch (error) {
      setHasil({
        kind: "conflict",
        pesan:
          error instanceof ApiError
            ? error.message
            : "Penugasan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      })
    } finally {
      setKirim(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={tutup}>
      <DialogTrigger render={<Button variant="outline" size={compact ? "xs" : "sm"} />}>
        <UserPlusIcon data-icon="inline-start" />
        Tugaskan
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {hasil?.kind === "ok" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                Inspektor ditugaskan
              </DialogTitle>
              <DialogDescription>
                {hasil.nama} ditugaskan pada {order.ref}. Klien melihat perubahan ini tanpa
                memuat ulang halaman, dan order muncul di daftar tugas inspektor.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : hasil?.kind === "conflict" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <TriangleAlertIcon className="size-4 text-attention" />
                Perubahan ditolak
              </DialogTitle>
              <DialogDescription>{hasil.pesan}</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Tugaskan inspektor untuk {order.ref}</DialogTitle>
              <DialogDescription>
                {order.location}, {order.city}. Urutan mengikuti jumlah penugasan yang sedang
                berjalan.
              </DialogDescription>
            </DialogHeader>

            <Select value={pilihan} onValueChange={(v) => setPilihan(String(v))}>
              <SelectTrigger className="w-full">
                <SelectValue>
                  {urut.find((i) => i.id === pilihan)?.name ?? "Pilih inspektor"}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {urut.map((i) => (
                  <SelectItem
                    key={i.id}
                    value={i.id}
                    label={`${i.name} · ${i.activeJobs} penugasan berjalan`}
                  >
                    {i.name} · {i.activeJobs} penugasan berjalan
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Batal</DialogClose>
              <Button onClick={tugaskan} disabled={kirim || !pilihan}>
                {kirim ? <Spinner /> : null}
                Tugaskan
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
