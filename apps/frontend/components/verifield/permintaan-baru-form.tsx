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
import { useActorHref } from "@/lib/actor/hooks"
import { useActor } from "@/components/verifield/actor-provider"
import { cn } from "@/lib/utils"
import { ApiError } from "@/lib/api/client"
import { createOrder } from "@/lib/api/orders"
import type { InspectionType, JobOrder } from "@/lib/domain/types"

/** Durasi baku satu penugasan. Backend menuntut rentang waktu, sedangkan klien
    hanya tahu kapan ia ingin pekerjaan dimulai — memaksanya menebak jam selesai
    hanya menambah satu isian yang jawabannya tidak ia miliki. */
const DURATION_HOURS = 6

/** Nilai datetime-local memakai waktu setempat, bukan ISO. */
function inputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** Jam kerja lapangan. Di luar rentang ini tidak ada inspektor maupun pihak
    ketiga di lokasi, sehingga jadwalnya tidak mungkin dieksekusi. */
const OPENS_AT_HOUR = 8
const CLOSES_AT_HOUR = 17

function withinBusinessHours(d: Date): boolean {
  const hour = d.getHours()
  return hour >= OPENS_AT_HOUR && hour <= CLOSES_AT_HOUR
}

export function PermintaanBaruForm({ types }: { types: InspectionType[] }) {
  const actor = useActor()
  const href = useActorHref()
  const router = useRouter()

  const [typeId, setTypeId] = React.useState(types[0]?.id ?? "")
  const [submitting, setSubmitting] = React.useState(false)
  const [result, setResult] = React.useState<JobOrder | null>(null)
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null)
  // Batas bawah jadwal dihitung sekali saat komponen hidup di browser. Saat
  // SSR nilainya kosong, dan selisih atribut min itu disembunyikan lewat
  // suppressHydrationWarning — menghitungnya di server hanya menghasilkan nilai
  // yang basi beberapa detik dan tetap bisa berselisih satu menit.
  const [minSchedule] = React.useState(() =>
    typeof window === "undefined" ? "" : inputValue(new Date()),
  )

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const start = new Date(String(form.get("jadwal")))

    if (!Number.isFinite(start.getTime()) || !withinBusinessHours(start)) {
      setErrorMessage("Jadwal hanya dapat dipilih pada jam kerja, pukul 08.00–17.00.")
      return
    }

    setSubmitting(true)
    setErrorMessage(null)
    try {
      const order = await createOrder(actor.id, {
        inspection_type_id: typeId,
        object_description: String(form.get("objek")),
        location_name: String(form.get("lokasi")),
        location_address: String(form.get("alamat")),
        city: String(form.get("kota")),
        scheduled_start_at: start.toISOString(),
        scheduled_end_at: new Date(
          start.getTime() + DURATION_HOURS * 3_600_000,
        ).toISOString(),
      })
      setResult(order)
      // Daftar order dirender di server; ia perlu diambil ulang agar order baru
      // ini sudah ada saat klien menekan "Lihat daftar order".
      router.refresh()
    } catch (error) {
      setErrorMessage(
        error instanceof ApiError
          ? error.message
          : "Permintaan tidak dapat dikirim. Periksa koneksi Anda lalu coba lagi.",
      )
    } finally {
      setSubmitting(false)
    }
  }

  if (result) {
    return (
      <Card className="mx-auto max-w-lg items-center gap-3 p-8 text-center">
        <CircleCheckIcon className="size-8 text-status-completed" />
        <h1 className="text-base font-semibold">Permintaan diterima</h1>
        <p className="text-sm leading-relaxed text-muted-foreground">
          Order Anda bernomor{" "}
          <strong className="tabular font-mono text-foreground">{result.ref}</strong> dan
          berstatus <strong className="font-medium text-foreground">Diminta</strong>.
          Koordinator akan menugaskan inspektor, dan statusnya berubah di layar Anda tanpa
          perlu memuat ulang halaman.
        </p>
        <Link href={href(`/klien/order/${result.id}`)} className={cn(buttonVariants(), "mt-2")}>
          Lihat order ini
        </Link>
      </Card>
    )
  }

  return (
    <div className="mx-auto max-w-lg">
      <Link
        href={href("/klien")}
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
              <Select value={typeId} onValueChange={(v) => setTypeId(String(v))}>
                <SelectTrigger id="jenis" className="w-full">
                  {/* Base UI mendaftarkan label lewat ref DOM, yang tidak ada saat
                      SSR — tanpa children ini pemicu menampilkan uuid mentah. */}
                  <SelectValue>{types.find((t) => t.id === typeId)?.name}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {types.map((t) => (
                    <SelectItem key={t.id} value={t.id} label={t.name}>
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
                min={minSchedule || undefined}
                suppressHydrationWarning
                required
              />
              <FieldDescription>
                Jadwal dapat bergeser mengikuti kesiapan pihak ketiga di lokasi. Pemilihan
                terbatas pada jam kerja, pukul 08.00–17.00.
              </FieldDescription>
            </Field>
          </FieldGroup>

          {errorMessage ? (
            <p className="mt-4 rounded-lg border border-destructive/30 bg-destructive/8 px-3 py-2 text-xs leading-relaxed">
              {errorMessage}
            </p>
          ) : null}

          <div className="mt-6 flex justify-end gap-2">
            <Link href={href("/klien")} className={buttonVariants({ variant: "secondary" })}>
              Batal
            </Link>
            <Button type="submit" disabled={submitting || !typeId}>
              {submitting ? <Spinner /> : null}
              Kirim Permintaan
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
