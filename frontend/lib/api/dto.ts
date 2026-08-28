import type { Role, Status } from "@/lib/domain/types"

/** Bentuk kawat — snake_case, persis seperti yang dikirim Go BE. Dipisahkan dari
    tipe domain supaya komponen tidak pernah terikat pada penamaan backend;
    seluruh perbedaannya diserap mapper di orders.ts. */

export type FieldError = { field: string; message: string }

export type StatusEventDTO = {
  id: string
  seq: number
  job_order_id: string
  from_status: Status | null
  to_status: Status
  actor_id: string | null
  actor_name: string
  actor_role: Role
  occurred_at: string
  received_at: string
  occurred_at_adjusted: boolean
  client_event_id: string | null
  accepted: boolean
  rejection_reason: string | null
  is_correction: boolean
  reason: string | null
  created_at: string
}

export type PendingCancellationDTO = {
  id: string
  reason: string
  requested_by_name: string
  created_at: string
}

export type JobOrderDTO = {
  id: string
  reference_number: string
  company_id: string
  company_name: string
  created_by_name: string
  inspection_type_id: string
  inspection_type_name: string
  object_description: string
  location_name: string
  location_address: string
  city: string
  scheduled_start_at: string
  scheduled_end_at: string
  inspector_id: string | null
  inspector_name: string | null
  current_status: Status
  status_changed_at: string
  version: number
  seq: number
  cancellation_requested: boolean
  has_open_alert: boolean
  exit_status: Status | null
  pending_cancellation?: PendingCancellationDTO
  events?: StatusEventDTO[]
  created_at: string
  updated_at: string
}

export type InspectorDTO = {
  id: string
  name: string
  email: string
  active_jobs: number
}

export type InspectionTypeDTO = {
  id: string
  code: string
  name: string
}

export type ActorDTO = {
  id: string
  name: string
  email: string
  role: Role
  company_id: string | null
  company_name: string | null
}

/** Hasil pembaruan status dari lapangan. Ditolak pun tetap 200 — yang gagal
    adalah perubahan statusnya, bukan pengirimannya (B-07). */
export type SubmitEventResultDTO = {
  accepted: boolean
  duplicate: boolean
  message: string
  event: StatusEventDTO
  order: JobOrderDTO
}

export type CancelResultDTO = {
  status: "cancelled" | "pending_approval"
  fee: string
  message: string
  order: JobOrderDTO
}
