"use client"

import * as React from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowLeftIcon, CircleCheckIcon } from "lucide-react"

import { Button, buttonVariants } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Spinner } from "@/components/ui/spinner"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useActor } from "@/components/verifield/actor-provider"
import { cn } from "@/lib/utils"
import { ApiError } from "@/lib/api/client"
import { createOrder } from "@/lib/api/orders"
import type { InspectionType, JobOrder } from "@/lib/domain/types"

/** Durasi baku satu penugasan. Backend menuntut rentang waktu, sedangkan klien
    hanya tahu kapan ia ingin pekerjaan dimulai — memaksanya menebak jam selesai
    hanya menambah satu isian yang jawabannya tidak ia miliki. */
const DURASI_JAM = 6

/** Nilai datetime-local memakai waktu setempat, bukan ISO. */
function inputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function PermintaanBaruForm({ types }: { types: InspectionType[] }) {
  const actor = useActor()
  const router = useRouter()

  const [jenis, setJenis] = React.useState(types[0]?.id ?? "")
  const [kirim, setKirim] = React.useState(false)
  const [hasil, setHasil] = React.useState<JobOrder | null>(null)
  const [galat, setGalat] = React.useState<string | null>(null)
  // Batas bawah jadwal dihitung sekali saat komponen hidup di browser. Saat
  // SSR nilainya kosong, dan selisih atribut min itu disembunyikan lewat
  // suppressHydrationWarning — menghitungnya di server hanya menghasilkan nilai
  // yang basi beberapa detik dan tetap bisa berselisih satu menit.
  const [minJadwal] = React.useState(() =>
    typeof window === "undefined" ? "" : inputValue(new Date()),
  )

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const mulai = new Date(String(form.get("jadwal")))

    setKirim(true)
    setGalat(null)
    try {
      const order = await createOrder(actor.id, {
        inspection_type_id: jenis,
        object_description: String(form.get("objek")),
        location_name: String(form.get("lokasi")),
        location_address: String(form.get("alamat")),
        city: String(form.get("kota")),
        scheduled_start_at: mulai.toISOString(),
        scheduled_end_at: new Date(
          mulai.getTime() + DURASI_JAM * 3_600_000,
        ).toISOString(),
      })
      setHasil(order)
      // Daftar order dirender di server; ia perlu diambil ulang agar order baru
      // ini sudah ada saat klien menekan "Lihat daftar order".
      router.refresh()
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

  if (hasil) {
    return (
      <Card className="mx-auto max-w-lg items-center gap-3 p-8 text-center">
        <CircleCheckIcon className="size-8 text-status-completed" />
        <h1 className="text-base font-semibold">Permintaan diterima</h1>
        <p className="text-sm leading-relaxed text-muted-foreground">
          Order Anda bernomor{" "}
          <strong className="tabular font-mono text-foreground">{hasil.ref}</strong> dan
          berstatus <strong className="font-medium text-foreground">Diminta</strong>.
          Koordinator akan menugaskan inspektor, dan statusnya berubah di layar Anda tanpa
          perlu memuat ulang halaman.
        </p>
        <Link href={`/klien/order/${hasil.id}`} className={cn(buttonVariants(), "mt-2")}>
          Lihat order ini
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
        <form onSubmit={submit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="jenis">Jenis inspeksi</FieldLabel>
              <Select value={jenis} onValueChange={(v) => setJenis(String(v))}>
                <SelectTrigger id="jenis">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {types.map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="objek">Objek yang diperiksa</FieldLabel>
              <Input id="objek" name="objek" required placeholder="Contoh: Kargo curah 8.500 MT" />
              <FieldDescription>
                Sebutkan jenis dan kuantitasnya agar inspektor dapat menyiapkan peralatan.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="lokasi">Lokasi</FieldLabel>
              <Input
                id="lokasi"
                name="lokasi"
                required
                placeholder="Contoh: Dermaga 3, Pelabuhan Tanjung Priok"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="alamat">Alamat lengkap</FieldLabel>
              <Textarea
                id="alamat"
                name="alamat"
                rows={2}
                required
                placeholder="Contoh: Jl. Palmerah No. 1, Tanjung Priok, Jakarta Utara"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="kota">Kota</FieldLabel>
              <Input id="kota" name="kota" required placeholder="Contoh: Jakarta" />
            </Field>

            <Field>
              <FieldLabel htmlFor="jadwal">Jadwal yang diminta</FieldLabel>
              <Input
                id="jadwal"
                name="jadwal"
                type="datetime-local"
                min={minJadwal || undefined}
                suppressHydrationWarning
                required
              />
              <FieldDescription>
                Jadwal dapat bergeser mengikuti kesiapan pihak ketiga di lokasi.
              </FieldDescription>
            </Field>
          </FieldGroup>

          {galat ? (
            <p className="mt-4 rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
              {galat}
            </p>
          ) : null}

          <div className="mt-6 flex justify-end gap-2">
            <Link href="/klien" className={buttonVariants({ variant: "secondary" })}>
              Batal
            </Link>
            <Button type="submit" disabled={kirim || !jenis}>
              {kirim ? <Spinner /> : null}
              Kirim Permintaan
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
