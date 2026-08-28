import type { Role, Status } from "@verifield/contract"

/** Status dan peran TIDAK ditulis ulang di sini — keduanya berasal dari anotasi
    Go lewat paket kontrak. Menyalinnya berarti dua daftar yang bisa menyimpang
    diam-diam; dengan cara ini, menambah satu status di backend langsung membuat
    setiap switch yang belum menanganinya gagal dikompilasi. */
export type { Role, Status }

/** Segmen URL yang mewakili peran. Ini yang dilihat pengguna; Role yang dipakai
    aturan bisnis. */
export type Persona = "klien" | "ops" | "lapangan"

export const PERSONA_ROLE: Record<Persona, Role> = {
  klien: "client",
  ops: "admin",
  lapangan: "inspector",
}

/** Satu entri riwayat bisa berarti enam hal yang berbeda. Backend menyimpannya
    sebagai kombinasi is_correction, accepted, dan rejection_reason; nilai ini
    adalah bentuk turunannya yang lebih mudah dibaca komponen. */
export type EventKind =
  | "transition"
  | "correction"
  | "late_rejected"
  | "out_of_order"
  | "cancellation_request"
  | "cancellation_rejected"
  | "cancellation_obsolete"

export type StatusEvent = {
  id: string
  seq: number
  orderId: string
  kind: EventKind
  from: Status | null
  to: Status
  /** Waktu kejadian di lapangan — inilah yang sah untuk laporan dan penagihan (B-02). */
  eventTime: string
  /** Waktu sistem menerima. Bisa berjam-jam setelah eventTime di area tanpa sinyal (B-02). */
  receivedTime: string
  /** Waktu yang dilaporkan perangkat berada di luar batas wajar dan tidak dipakai. */
  timeAdjusted: boolean
  actorRole: Role
  actorName: string
  reason?: string
  /** Dibuat di perangkat, sebelum kiriman pertama. Kiriman berulang dengan penanda
      sama diabaikan, bukan dicatat dua kali (B-03). */
  idempotencyKey: string | null
}

export type PendingCancellation = {
  id: string
  reason: string
  requestedByName: string
  createdAt: string
}

export type JobOrder = {
  id: string
  ref: string
  clientName: string
  inspectionType: string
  object: string
  location: string
  address: string
  city: string
  scheduledAt: string
  scheduledEndAt: string
  inspectorId: string | null
  inspectorName: string | null
  status: Status
  statusChangedAt: string
  /** Tahap terjauh yang sempat dicapai. Untuk order yang berakhir sebelum
      selesai, inilah yang menjawab "sejauh mana sempat berjalan" — dihitung
      server karena daftar order sengaja tidak membawa riwayat. */
  exitStatus: Status | null
  /** Dikirim balik saat mutasi sebagai expectedVersion. Tanpa ini, perubahan
      koordinator kedua menimpa yang pertama tanpa ada yang sadar (B-09). */
  version: number
  /** Seq event terakhir milik order ini. Pesan real-time yang seq-nya tidak lebih
      besar dari ini diabaikan, sehingga kiriman ganda maupun yang datang terbalik
      urutannya tidak pernah membuat layar mundur. */
  seq: number
  cancellationRequested: boolean
  hasOpenAlert: boolean
  pendingCancellation: PendingCancellation | null
  /** Hanya terisi pada halaman detail; daftar tidak membawa riwayat. */
  events: StatusEvent[]
  createdAt: string
  updatedAt: string
}

export type Inspector = {
  id: string
  name: string
  email: string
  activeJobs: number
}

export type InspectionType = {
  id: string
  code: string
  name: string
}

export type Actor = {
  id: string
  name: string
  email: string
  role: Role
  companyId: string | null
  companyName: string | null
}
