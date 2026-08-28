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

type Outcome = { kind: "ok"; name: string } | { kind: "conflict"; message: string } | null

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
  const applyToStore = useApplyResult()

  const [open, setOpen] = React.useState(false)
  const [submitting, setSubmitting] = React.useState(false)
  const [result, setResult] = React.useState<Outcome>(null)

  // Beban kerja jadi dasar urutan karena penugasan otomatis berada di luar
  // cakupan — angka inilah yang menggantikan pertimbangan itu.
  const ordered = React.useMemo(
    () => [...inspectors].sort((a, b) => a.activeJobs - b.activeJobs),
    [inspectors],
  )
  const [options, setPilihan] = React.useState(ordered[0]?.id ?? "")

  function handleOpenChange(next: boolean) {
    setOpen(next)
    if (!next) setResult(null)
  }

  async function tugaskan() {
    setSubmitting(true)
    try {
      const latest = await assignInspector(actor.id, order.id, options, order.version)
      applyToStore(latest)
      setResult({ kind: "ok", name: latest.inspectorName ?? "Inspektor" })
    } catch (error) {
      setResult({
        kind: "conflict",
        message:
          error instanceof ApiError
            ? error.message
            : "Penugasan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={<Button variant="outline" size={compact ? "xs" : "sm"} />}>
        <UserPlusIcon data-icon="inline-start" />
        Tugaskan
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {result?.kind === "ok" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                Inspektor ditugaskan
              </DialogTitle>
              <DialogDescription>
                {result.name} ditugaskan pada {order.ref}. Klien melihat perubahan ini tanpa
                memuat ulang halaman, dan order muncul di daftar tugas inspektor.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : result?.kind === "conflict" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <TriangleAlertIcon className="size-4 text-attention" />
                Perubahan ditolak
              </DialogTitle>
              <DialogDescription>{result.message}</DialogDescription>
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

            <Select value={options} onValueChange={(v) => setPilihan(String(v))}>
              <SelectTrigger className="w-full">
                <SelectValue>
                  {ordered.find((i) => i.id === options)?.name ?? "Pilih inspektor"}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {ordered.map((i) => (
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
              <Button onClick={tugaskan} disabled={submitting || !options}>
                {submitting ? <Spinner /> : null}
                Tugaskan
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
