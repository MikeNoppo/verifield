/** Zona waktu ditetapkan eksplisit, bukan mengikuti zona pembaca. Selain karena
    operasi diasumsikan satu zona (A-04), ini juga membuat keluaran server dan
    klien identik sehingga tidak ada hydration mismatch. */
const TZ = "Asia/Jakarta"

const jam = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
})

const tanggalPendek = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  day: "numeric",
  month: "short",
})

const tanggalPanjang = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  day: "numeric",
  month: "long",
  year: "numeric",
})

export function waktu(iso: string): string {
  return jam.format(new Date(iso)).replace(":", ".")
}

export function tanggal(iso: string): string {
  return tanggalPendek.format(new Date(iso))
}

export function tanggalJam(iso: string): string {
  return `${tanggalPendek.format(new Date(iso))} ${waktu(iso)}`
}

export function tanggalLengkap(iso: string): string {
  return `${tanggalPanjang.format(new Date(iso))}, ${waktu(iso)}`
}

export function durasi(ms: number): string {
  const menit = Math.round(ms / 60_000)
  if (menit < 1) return "kurang dari semenit"
  if (menit < 60) return `${menit}m`
  const j = Math.floor(menit / 60)
  const m = menit % 60
  if (j < 24) return m === 0 ? `${j}j` : `${j}j ${m}m`
  const h = Math.floor(j / 24)
  return `${h}h ${j % 24}j`
}

export function selisih(dari: string, sampai: string): number {
  return new Date(sampai).getTime() - new Date(dari).getTime()
}

export function sejak(iso: string, now: Date): string {
  const ms = now.getTime() - new Date(iso).getTime()
  if (ms < 60_000) return "baru saja"
  return `${durasi(ms)} lalu`
}
