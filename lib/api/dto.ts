import type { EventKind, Role, Status } from "@/lib/domain/types"

/** Bentuk kawat — snake_case, persis seperti yang akan dikirim Go BE. Dipisahkan
    dari tipe domain supaya komponen tidak pernah terikat pada penamaan BE, dan
    penggantian mock ke API asli cukup menyentuh mapper di orders.ts. */

export type FieldError = { field: string; message: string }

export type StatusEventDTO = {
  id: string
  order_id: string
  kind: EventKind
  from_status: Status | null
  to_status: Status
  event_time: string
  received_time: string
  actor_role: Role
  actor_name: string
  reason?: string
  idempotency_key: string
}

export type JobOrderDTO = {
  id: string
  ref: string
  client_name: string
  inspection_type: string
  object: string
  location: string
  city: string
  scheduled_at: string
  inspector_id: string | null
  inspector_name: string | null
  status: Status
  version: number
  cancellation_requested: boolean
  events: StatusEventDTO[]
  created_at: string
  updated_at: string
}

export type InspectorDTO = {
  id: string
  name: string
  code: string
  city: string
  active_jobs: number
}
