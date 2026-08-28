export type Status =
  | "requested"
  | "assigned"
  | "on_the_way"
  | "on_site"
  | "in_progress"
  | "completed"
  | "failed"
  | "cancelled"

export type Role = "klien" | "ops" | "lapangan"

export type EventKind =
  | "transition"
  | "correction"
  | "late_rejected"
  | "cancellation_request"

export type StatusEvent = {
  id: string
  orderId: string
  kind: EventKind
  from: Status | null
  to: Status
  /** Waktu kejadian di lapangan — inilah yang sah untuk laporan dan penagihan (B-02). */
  eventTime: string
  /** Waktu sistem menerima. Bisa berjam-jam setelah eventTime di area tanpa sinyal (B-02). */
  receivedTime: string
  actorRole: Role
  actorName: string
  reason?: string
  /** Dibuat di perangkat, sebelum kiriman pertama. Kiriman berulang dengan penanda
      sama diabaikan, bukan dicatat dua kali (B-03). */
  idempotencyKey: string
}

export type JobOrder = {
  id: string
  ref: string
  clientName: string
  inspectionType: string
  object: string
  location: string
  city: string
  scheduledAt: string
  inspectorId: string | null
  inspectorName: string | null
  status: Status
  /** Dikirim balik saat mutasi sebagai expectedVersion. Tanpa ini, perubahan
      koordinator kedua menimpa yang pertama tanpa ada yang sadar (B-09). */
  version: number
  cancellationRequested: boolean
  events: StatusEvent[]
  createdAt: string
  updatedAt: string
}

export type Inspector = {
  id: string
  name: string
  code: string
  city: string
  activeJobs: number
}
