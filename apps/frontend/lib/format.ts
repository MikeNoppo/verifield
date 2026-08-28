/** Zona waktu ditetapkan eksplisit, bukan mengikuti zona pembaca. Selain karena
    operasi diasumsikan satu zona (A-04), ini juga membuat keluaran server dan
    klien identik sehingga tidak ada hydration mismatch. */
const TZ = "Asia/Jakarta"

const clock = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
})

const shortDate = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  day: "numeric",
  month: "short",
})

const longDate = new Intl.DateTimeFormat("id-ID", {
  timeZone: TZ,
  day: "numeric",
  month: "long",
  year: "numeric",
})

export function formatTime(iso: string): string {
  return clock.format(new Date(iso)).replace(":", ".")
}

export function formatDate(iso: string): string {
  return shortDate.format(new Date(iso))
}

export function formatDateTime(iso: string): string {
  return `${shortDate.format(new Date(iso))} ${formatTime(iso)}`
}

export function formatFullDateTime(iso: string): string {
  return `${longDate.format(new Date(iso))}, ${formatTime(iso)}`
}

export function formatDuration(ms: number): string {
  const minutes = Math.round(ms / 60_000)
  if (minutes < 1) return "kurang dari semenit"
  if (minutes < 60) return `${minutes}m`

  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  if (hours < 24) return restMinutes === 0 ? `${hours}j` : `${hours}j ${restMinutes}m`

  const days = Math.floor(hours / 24)
  return `${days}h ${hours % 24}j`
}

export function millisBetween(from: string, to: string): number {
  return new Date(to).getTime() - new Date(from).getTime()
}

export function relativeTime(iso: string, now: Date): string {
  const ms = now.getTime() - new Date(iso).getTime()
  if (ms < 60_000) return "baru saja"
  return `${formatDuration(ms)} lalu`
}
