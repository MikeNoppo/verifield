import type { JobOrder, Role, Status } from "./types"

/** Enam tahap yang dilalui pekerjaan normal. Terminal Failed dan Cancelled
    berada di luar rel ini — keduanya keluar dari tahap mana pun. */
export const PIPELINE = [
  "requested",
  "assigned",
  "on_the_way",
  "on_site",
  "in_progress",
  "completed",
] as const satisfies readonly Status[]

export const TERMINAL: readonly Status[] = ["completed", "failed", "cancelled"]

export const STATUS_LABEL: Record<Status, string> = {
  requested: "Diminta",
  assigned: "Ditugaskan",
  on_the_way: "Dalam Perjalanan",
  on_site: "Di Lokasi",
  in_progress: "Sedang Dikerjakan",
  completed: "Selesai",
  failed: "Gagal",
  cancelled: "Dibatalkan",
}

export const STATUS_MEANING: Record<Status, string> = {
  requested: "Order diterima, inspektor belum ditentukan",
  assigned: "Inspektor sudah ditetapkan, belum berangkat",
  on_the_way: "Inspektor dalam perjalanan menuju lokasi",
  on_site: "Inspektor tiba di lokasi, pekerjaan belum dimulai",
  in_progress: "Pemeriksaan atau sampling sedang berlangsung",
  completed: "Pekerjaan lapangan selesai",
  failed: "Inspektor tiba, pekerjaan tidak dapat dilaksanakan",
  cancelled: "Order dibatalkan sebelum atau saat pelaksanaan",
}

export function isTerminal(status: Status): boolean {
  return TERMINAL.includes(status)
}

export function pipelineIndex(status: Status): number {
  return (PIPELINE as readonly Status[]).indexOf(status)
}

/** Tahap tempat pekerjaan berhenti ketika berakhir Failed atau Cancelled —
    dibaca dari transisi terakhir sebelum terminal, bukan ditebak. */
export function exitIndex(order: JobOrder): number {
  const lastBefore = [...order.events]
    .reverse()
    .find((e) => e.kind === "transition" && !isTerminal(e.to))
  return lastBefore ? pipelineIndex(lastBefore.to) : 0
}

const FORWARD: Record<Status, readonly Status[]> = {
  requested: ["assigned", "cancelled"],
  assigned: ["on_the_way", "cancelled"],
  on_the_way: ["on_site", "cancelled"],
  // Failed hanya mungkin setelah inspektor benar-benar tiba — definisinya
  // "inspektor tiba, pekerjaan tidak dapat dilaksanakan".
  on_site: ["in_progress", "failed", "cancelled"],
  in_progress: ["completed", "failed", "cancelled"],
  completed: [],
  failed: [],
  cancelled: [],
}

/** Maju saja. Koreksi mundur bukan transisi — itu wewenang koordinator lewat
    jalur terpisah yang wajib beralasan (B-06). */
export function canTransition(from: Status, to: Status): boolean {
  return FORWARD[from].includes(to)
}

export type CancelFee = "none" | "travel" | "visit" | "coordinator"

export type CancelAuthority =
  | { kind: "allowed"; fee: CancelFee; consequence: string }
  | { kind: "requires_approval"; message: string }
  | { kind: "forbidden"; message: string }

const FEE_TEXT: Record<CancelFee, string> = {
  none: "Tidak ada biaya pada tahap ini.",
  travel: "Inspektor sudah dalam perjalanan, sehingga dikenakan biaya perjalanan.",
  visit: "Inspektor sudah tiba di lokasi, sehingga dikenakan biaya kunjungan.",
  coordinator: "Biaya ditentukan koordinator.",
}

/** Matriks kewenangan pembatalan. Bukan boolean: tahap In Progress menghasilkan
    permintaan yang menunggu keputusan koordinator, bukan pembatalan langsung
    (B-05), dan setiap penolakan membawa kalimatnya sendiri (F-05). */
export function cancelAuthority(role: Role, status: Status): CancelAuthority {
  if (isTerminal(status)) {
    return {
      kind: "forbidden",
      message: `Order ini sudah ${STATUS_LABEL[status].toLowerCase()} dan tidak dapat diubah lagi.`,
    }
  }

  if (role === "inspector") {
    return {
      kind: "forbidden",
      message:
        "Inspektor tidak berwenang membatalkan order. Bila pekerjaan tidak dapat dilaksanakan, laporkan sebagai kendala beserta alasannya.",
    }
  }

  if (status === "in_progress" && role === "client") {
    return {
      kind: "requires_approval",
      message:
        "Pekerjaan sudah dimulai di lokasi. Permintaan pembatalan Anda akan kami teruskan ke koordinator untuk ditinjau.",
    }
  }

  const fee: CancelFee =
    status === "on_the_way"
      ? "travel"
      : status === "on_site"
        ? "visit"
        : status === "in_progress"
          ? "coordinator"
          : "none"

  return { kind: "allowed", fee, consequence: FEE_TEXT[fee] }
}

export type Action = {
  to: Status
  label: string
  /** Aksi utama muncul sebagai tombol besar; sekunder sengaja dibuat kecil dan
      butuh langkah tambahan. */
  tone: "primary" | "secondary"
}

/** Aksi berikutnya bergantung pada peran DAN status. Inspektor tidak pernah
    memilih dari daftar — ia hanya menekan satu tombol yang sudah benar
    secara konteks (F-04). */
export function inspectorActions(status: Status): Action[] {
  switch (status) {
    case "assigned":
      return [{ to: "on_the_way", label: "Berangkat", tone: "primary" }]
    case "on_the_way":
      return [{ to: "on_site", label: "Tiba di Lokasi", tone: "primary" }]
    case "on_site":
      return [
        { to: "in_progress", label: "Mulai Bekerja", tone: "primary" },
        { to: "failed", label: "Laporkan Kendala", tone: "secondary" },
      ]
    case "in_progress":
      return [
        { to: "completed", label: "Selesai", tone: "primary" },
        { to: "failed", label: "Laporkan Kendala", tone: "secondary" },
      ]
    default:
      return []
  }
}

/** Kabar terakhir dari lapangan sudah terlalu lama. Dihitung dari waktu status
    berubah, bukan waktu baris diperbarui — laporan yang tertahan tiga jam di
    perangkat memang membuat order langsung tampak basi, dan itu benar. */
export function isStale(order: JobOrder, now: Date, hours = 8): boolean {
  if (isTerminal(order.status)) return false
  const last = new Date(order.statusChangedAt).getTime()
  return now.getTime() - last > hours * 3_600_000
}

/** Dibaca dari tanda yang dihitung server, bukan dari riwayat — daftar order
    sengaja tidak membawa riwayat, dan justru di daftar itulah tanda ini paling
    dibutuhkan koordinator. */
export function hasLateRejected(order: JobOrder): boolean {
  return order.hasOpenAlert
}

export function needsAssignment(order: JobOrder): boolean {
  return order.status === "requested"
}
