import { apiFetch, apiFetchList, toQueryString, type ListQuery } from "./client"
import type {
  ActorDTO,
  CancelResultDTO,
  InspectionTypeDTO,
  InspectorDTO,
  JobOrderDTO,
  StatusEventDTO,
  SubmitEventResultDTO,
} from "@verifield/contract"
import type {
  Actor,
  EventKind,
  InspectionType,
  Inspector,
  JobOrder,
  Status,
  StatusEvent,
} from "@/lib/domain/types"

// ---------------------------------------------------------------------------
// Mapper
// ---------------------------------------------------------------------------

/** Tipe kawat berasal dari anotasi Go lewat @verifield/contract, dan di sana
    field yang tidak ada bernilai undefined. Tipe domain memakai null. Mapper di
    berkas inilah satu-satunya tempat kedua kosakata itu bertemu. */

/** Backend menyimpan jenis entri sebagai kombinasi tiga kolom karena itulah
    bentuk yang benar untuk disimpan. Komponen hanya perlu satu nilai untuk
    memutuskan cara menggambarnya. */
function kindOf(d: StatusEventDTO): EventKind {
  if (d.is_correction) return "correction"
  if (d.accepted) return "transition"

  switch (d.rejection_reason) {
    case "late_after_final":
      return "late_rejected"
    case "out_of_order":
      return "out_of_order"
    case "pending_approval":
      return "cancellation_request"
    case "cancellation_rejected":
      return "cancellation_rejected"
    default:
      return "transition"
  }
}

export function eventFromDTO(d: StatusEventDTO): StatusEvent {
  return {
    id: d.id,
    seq: d.seq,
    orderId: d.job_order_id,
    kind: kindOf(d),
    from: d.from_status ?? null,
    to: d.to_status,
    eventTime: d.occurred_at,
    receivedTime: d.received_at,
    timeAdjusted: d.occurred_at_adjusted,
    actorRole: d.actor_role,
    actorName: d.actor_name,
    reason: d.reason ?? undefined,
    idempotencyKey: d.client_event_id ?? null,
  }
}

export function orderFromDTO(d: JobOrderDTO): JobOrder {
  return {
    id: d.id,
    ref: d.reference_number,
    clientName: d.company_name,
    inspectionType: d.inspection_type_name,
    object: d.object_description,
    location: d.location_name,
    address: d.location_address,
    city: d.city,
    scheduledAt: d.scheduled_start_at,
    scheduledEndAt: d.scheduled_end_at,
    inspectorId: d.inspector_id ?? null,
    inspectorName: d.inspector_name ?? null,
    status: d.current_status,
    statusChangedAt: d.status_changed_at,
    version: d.version,
    seq: d.seq,
    cancellationRequested: d.cancellation_requested,
    hasOpenAlert: d.has_open_alert,
    exitStatus: d.exit_status ?? null,
    pendingCancellation: d.pending_cancellation
      ? {
          id: d.pending_cancellation.id,
          reason: d.pending_cancellation.reason,
          requestedByName: d.pending_cancellation.requested_by_name,
          createdAt: d.pending_cancellation.created_at,
        }
      : null,
    events: (d.events ?? []).map(eventFromDTO),
    createdAt: d.created_at,
    updatedAt: d.updated_at,
  }
}

// ---------------------------------------------------------------------------
// Baca
// ---------------------------------------------------------------------------

export async function listOrders(
  actorId: string,
  query: ListQuery = {},
): Promise<{ orders: JobOrder[]; total: number }> {
  const { data, total } = await apiFetchList<JobOrderDTO[]>(
    `/orders${toQueryString(query)}`,
    { actorId },
  )
  return { orders: data.map(orderFromDTO), total }
}

export async function getOrder(actorId: string, id: string): Promise<JobOrder> {
  return orderFromDTO(await apiFetch<JobOrderDTO>(`/orders/${id}`, { actorId }))
}

/** Riwayat sejak kursor tertentu. Layar detail memakainya untuk menambahkan
    entri baru tanpa memuat ulang seluruh timeline — pesan real-time sengaja
    tidak membawa riwayat agar ukurannya tidak tumbuh seiring umur order. */
export async function listEvents(
  actorId: string,
  id: string,
  afterSeq = 0,
): Promise<StatusEvent[]> {
  const events = await apiFetch<StatusEventDTO[]>(
    `/orders/${id}/events?after_seq=${afterSeq}`,
    { actorId },
  )
  return events.map(eventFromDTO)
}

export async function listInspectors(): Promise<Inspector[]> {
  const data = await apiFetch<InspectorDTO[]>("/inspectors")
  return data.map((i) => ({
    id: i.id,
    name: i.name,
    email: i.email,
    activeJobs: i.active_jobs,
  }))
}

export async function listInspectionTypes(): Promise<InspectionType[]> {
  return apiFetch<InspectionTypeDTO[]>("/inspection-types")
}

export async function listActors(): Promise<Actor[]> {
  const data = await apiFetch<ActorDTO[]>("/demo/actors")
  return data.map((a) => ({
    id: a.id,
    name: a.name,
    email: a.email,
    role: a.role,
    companyId: a.company_id ?? null,
    companyName: a.company_name ?? null,
  }))
}

// ---------------------------------------------------------------------------
// Tulis
// ---------------------------------------------------------------------------

export type CreateOrderInput = {
  inspection_type_id: string
  object_description: string
  location_name: string
  location_address: string
  city: string
  scheduled_start_at: string
  scheduled_end_at: string
}

export async function createOrder(
  actorId: string,
  input: CreateOrderInput,
): Promise<JobOrder> {
  return orderFromDTO(
    await apiFetch<JobOrderDTO>("/orders", {
      actorId,
      method: "POST",
      body: JSON.stringify(input),
    }),
  )
}

export async function assignInspector(
  actorId: string,
  id: string,
  inspectorId: string,
  expectedVersion: number,
): Promise<JobOrder> {
  return orderFromDTO(
    await apiFetch<JobOrderDTO>(`/orders/${id}/assign`, {
      actorId,
      method: "POST",
      body: JSON.stringify({
        inspector_id: inspectorId,
        expected_version: expectedVersion,
      }),
    }),
  )
}

export type SubmitEventInput = {
  to_status: Status
  /** Dibuat di perangkat sebelum kiriman pertama, sehingga tetap sama walau
      laporan dikirim ulang berkali-kali (B-03). */
  client_event_id: string
  /** Waktu tombol ditekan, bukan waktu terkirim (B-02). */
  occurred_at?: string
  reason?: string
}

export type SubmitEventResult = {
  accepted: boolean
  duplicate: boolean
  message: string
  event: StatusEvent
  order: JobOrder
}

export async function submitStatusEvent(
  actorId: string,
  id: string,
  input: SubmitEventInput,
): Promise<SubmitEventResult> {
  const result = await apiFetch<SubmitEventResultDTO>(`/orders/${id}/events`, {
    actorId,
    method: "POST",
    body: JSON.stringify(input),
  })

  return {
    accepted: result.accepted,
    duplicate: result.duplicate,
    message: result.message,
    event: eventFromDTO(result.event),
    order: orderFromDTO(result.order),
  }
}

export type CancelResult = {
  status: "cancelled" | "pending_approval"
  fee: string
  message: string
  order: JobOrder
}

export async function cancelOrder(
  actorId: string,
  id: string,
  reason: string,
  expectedVersion?: number,
): Promise<CancelResult> {
  const result = await apiFetch<CancelResultDTO>(`/orders/${id}/cancel`, {
    actorId,
    method: "POST",
    body: JSON.stringify({ reason, expected_version: expectedVersion }),
  })

  return { ...result, order: orderFromDTO(result.order) }
}

export async function decideCancellation(
  actorId: string,
  id: string,
  requestId: string,
  decision: "approve" | "reject",
  note?: string,
): Promise<JobOrder> {
  return orderFromDTO(
    await apiFetch<JobOrderDTO>(`/orders/${id}/cancellations/${requestId}/decide`, {
      actorId,
      method: "POST",
      body: JSON.stringify({ decision, note }),
    }),
  )
}

export async function correctStatus(
  actorId: string,
  id: string,
  toStatus: Status,
  reason: string,
  expectedVersion: number,
): Promise<JobOrder> {
  return orderFromDTO(
    await apiFetch<JobOrderDTO>(`/orders/${id}/corrections`, {
      actorId,
      method: "POST",
      body: JSON.stringify({
        to_status: toStatus,
        reason,
        expected_version: expectedVersion,
      }),
    }),
  )
}
