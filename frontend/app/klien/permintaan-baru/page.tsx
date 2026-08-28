"use client"

import * as React from "react"
import Link from "next/link"
import { ArrowLeftIcon, CircleCheckIcon } from "lucide-react"

import { Button, buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { Card } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const JENIS = [
  "Draft Survey",
  "Sampling Batu Bara",
  "Kalibrasi Tangki",
  "Pemeriksaan Kontainer",
  "Analisa Kargo Curah",
  "Verifikasi Stockpile",
]

export default function PermintaanBaru() {
  const [terkirim, setTerkirim] = React.useState(false)

  if (terkirim) {
    return (
      <Card className="mx-auto max-w-lg items-center gap-3 p-8 text-center">
        <CircleCheckIcon className="size-8 text-status-completed" />
        <h1 className="text-base font-semibold">Permintaan diterima</h1>
        <p className="text-sm leading-relaxed text-muted-foreground">
          Order Anda berstatus <strong className="font-medium text-foreground">Diminta</strong>.
          Koordinator akan menugaskan inspektor, dan statusnya berubah di layar Anda tanpa perlu
          memuat ulang halaman.
        </p>
        <Link href="/klien" className={cn(buttonVariants(), "mt-2")}>
          Lihat daftar order
        </Link>
      </Card>
    )
  }

  return (
    <div className="mx-auto max-w-lg">
      <Link
        href="/klien"
        className="mb-4 inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
      >
        <ArrowLeftIcon className="size-3.5" />
        Kembali ke daftar order
      </Link>

      <h1 className="mb-1 text-lg font-semibold tracking-tight">Permintaan Inspeksi Baru</h1>
      <p className="mb-5 text-xs leading-relaxed text-muted-foreground">
        Satu permintaan mencakup satu objek, di satu lokasi, pada satu rentang waktu.
      </p>

      <Card className="p-5">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            setTerkirim(true)
          }}
        >
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="jenis">Jenis inspeksi</FieldLabel>
              <Select defaultValue={JENIS[0]}>
                <SelectTrigger id="jenis">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {JENIS.map((j) => (
                    <SelectItem key={j} value={j}>
                      {j}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="objek">Objek yang diperiksa</FieldLabel>
              <Input id="objek" required placeholder="Contoh: Kargo curah 8.500 MT" />
              <FieldDescription>
                Sebutkan jenis dan kuantitasnya agar inspektor dapat menyiapkan peralatan.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="lokasi">Lokasi</FieldLabel>
              <Input id="lokasi" required placeholder="Contoh: Pelabuhan Tanjung Priok" />
            </Field>

            <Field>
              <FieldLabel htmlFor="jadwal">Jadwal yang diminta</FieldLabel>
              <Input id="jadwal" type="datetime-local" required />
              <FieldDescription>
                Jadwal dapat bergeser mengikuti kesiapan pihak ketiga di lokasi.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="catatan">Catatan</FieldLabel>
              <Textarea id="catatan" rows={3} placeholder="Opsional" />
            </Field>
          </FieldGroup>

          <div className="mt-6 flex justify-end gap-2">
            <Link href="/klien" className={buttonVariants({ variant: "secondary" })}>
              Batal
            </Link>
            <Button type="submit">Kirim Permintaan</Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
