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
import type { Inspector } from "@/lib/domain/types"

export function AssignDialog({
  orderRef,
  city,
  inspectors,
  /** Order yang sengaja dipakai memperagakan B-09: koordinator lain sudah
      menugaskan lebih dulu, sehingga perubahan kedua ditolak. */
  simulateConflict = false,
  compact = false,
}: {
  orderRef: string
  city: string
  inspectors: Inspector[]
  simulateConflict?: boolean
  compact?: boolean
}) {
  const [open, setOpen] = React.useState(false)
  const [hasil, setHasil] = React.useState<"idle" | "ok" | "conflict">("idle")

  const terdekat = [...inspectors].sort((a, b) => {
    const kota = Number(b.city === city) - Number(a.city === city)
    return kota !== 0 ? kota : a.activeJobs - b.activeJobs
  })
  const [pilihan, setPilihan] = React.useState(terdekat[0]?.id ?? "")

  function tutup(next: boolean) {
    setOpen(next)
    if (!next) setHasil("idle")
  }

  return (
    <Dialog open={open} onOpenChange={tutup}>
      <DialogTrigger
        render={<Button variant="outline" size={compact ? "xs" : "sm"} />}
      >
        <UserPlusIcon data-icon="inline-start" />
        Tugaskan
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        {hasil === "ok" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <CircleCheckIcon className="size-4 text-status-completed" />
                Inspektor ditugaskan
              </DialogTitle>
              <DialogDescription>
                {orderRef} sekarang berstatus Ditugaskan. Perubahan ini langsung terlihat oleh
                klien dan muncul pada daftar tugas inspektor.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Tutup</DialogClose>
            </DialogFooter>
          </>
        ) : hasil === "conflict" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <TriangleAlertIcon className="size-4 text-attention" />
                Data telah berubah
              </DialogTitle>
              <DialogDescription>
                Koordinator lain menugaskan Budi Santoso pada {orderRef} 12 detik lalu.
                Penugasan Anda tidak diterapkan.
              </DialogDescription>
            </DialogHeader>
            <p className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-xs leading-relaxed text-muted-foreground">
              Perubahan pertama yang menang. Menerima keduanya berarti salah satu hilang tanpa
              ada yang menyadarinya — dan data ini menjadi dasar dokumen komersial.
            </p>
            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>
                Muat ulang tampilan
              </DialogClose>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Tugaskan inspektor</DialogTitle>
              <DialogDescription>
                {orderRef} · {city}. Daftar diurutkan berdasarkan kota yang sama lalu beban
                aktif paling sedikit.
              </DialogDescription>
            </DialogHeader>

            <Select value={pilihan} onValueChange={(v) => setPilihan(String(v))}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {terdekat.map((i) => (
                  <SelectItem key={i.id} value={i.id}>
                    {i.name} · {i.city} · {i.activeJobs} job aktif
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <DialogFooter>
              <DialogClose render={<Button variant="secondary" />}>Batal</DialogClose>
              <Button onClick={() => setHasil(simulateConflict ? "conflict" : "ok")}>
                Tugaskan
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
