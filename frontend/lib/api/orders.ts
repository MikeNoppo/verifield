import "server-only"

import type { JobOrderDTO, StatusEventDTO } from "./dto"
import type { Inspector, JobOrder, Status } from "@/lib/domain/types"
import { hasLateRejected, isStale, isTerminal, needsAssignment } from "@/lib/domain/status"
import { CURRENT_CLIENT, DEMO_NOW, INSPECTORS, ORDERS } from "@/lib/mock/seed"

/** Titik tukar ketika BE asli siap: ganti isi fungsi-fungsi di bawah dengan
    apiFetch, lalu petakan hasilnya lewat mapper ini. Komponen tidak berubah
    sama sekali karena tidak ada satu pun yang menyentuh penamaan snake_case. */
export function eventFromDTO(d: StatusEventDTO) {
  return {
    id: d.id,
    orderId: d.order_id,
    kind: d.kind,
    from: d.from_status,
    to: d.to_status,
    eventTime: d.event_time,
    receivedTime: d.received_time,
    actorRole: d.actor_role,
    actorName: d.actor_name,
    reason: d.reason,
    idempotencyKey: d.idempotency_key,
  }
}

export function orderFromDTO(d: JobOrderDTO): JobOrder {
  return {
    id: d.id,
    ref: d.ref,
    clientName: d.client_name,
    inspectionType: d.inspection_type,
    object: d.object,
    location: d.location,
    city: d.city,
    scheduledAt: d.scheduled_at,
    inspectorId: d.inspector_id,
    inspectorName: d.inspector_name,
    status: d.status,
    version: d.version,
    cancellationRequested: d.cancellation_requested,
    events: d.events.map(eventFromDTO),
    createdAt: d.created_at,
    updatedAt: d.updated_at,
  }
}

export type OrderFilter = {
  status?: Status
  bucket?: "berjalan" | "selesai" | "bermasalah"
  attention?: "penugasan" | "pembatalan" | "basi" | "terlambat"
  search?: string
}

function matchesBucket(o: JobOrder, bucket: OrderFilter["bucket"]): boolean {
  if (!bucket) return true
  if (bucket === "berjalan") return !isTerminal(o.status)
  if (bucket === "selesai") return o.status === "completed"
  return o.status === "failed" || o.status === "cancelled" || hasLateRejected(o)
}

function matchesAttention(o: JobOrder, a: OrderFilter["attention"]): boolean {
  if (!a) return true
  if (a === "penugasan") return needsAssignment(o)
  if (a === "pembatalan") return o.cancellationRequested
  if (a === "basi") return isStale(o, DEMO_NOW)
  return hasLateRejected(o)
}

function matchesSearch(o: JobOrder, q?: string): boolean {
  if (!q) return true
  const n = q.toLowerCase()
  return [o.ref, o.clientName, o.object, o.location, o.city, o.inspectorName ?? ""]
    .join(" ")
    .toLowerCase()
    .includes(n)
}

function sortByUrgency(a: JobOrder, b: JobOrder): number {
  const rank = (o: JobOrder) =>
    isTerminal(o.status) ? 2 : needsAssignment(o) || o.cancellationRequested ? 0 : 1
  const d = rank(a) - rank(b)
  if (d !== 0) return d
  return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
}

export function listOrders(filter: OrderFilter = {}): JobOrder[] {
  return ORDERS.filter(
    (o) =>
      (!filter.status || o.status === filter.status) &&
      matchesBucket(o, filter.bucket) &&
      matchesAttention(o, filter.attention) &&
      matchesSearch(o, filter.search),
  ).sort(sortByUrgency)
}

export function listClientOrders(filter: OrderFilter = {}): JobOrder[] {
  return listOrders(filter).filter((o) => o.clientName === CURRENT_CLIENT)
}

export function listInspectorTasks(inspectorId: string): JobOrder[] {
  return ORDERS.filter((o) => o.inspectorId === inspectorId && !isTerminal(o.status)).sort(
    (a, b) => new Date(a.scheduledAt).getTime() - new Date(b.scheduledAt).getTime(),
  )
}

export function listInspectorHistory(inspectorId: string): JobOrder[] {
  return ORDERS.filter((o) => o.inspectorId === inspectorId && isTerminal(o.status)).sort(
    (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
  )
}

export function getOrder(id: string): JobOrder | null {
  return ORDERS.find((o) => o.id === id.toLowerCase()) ?? null
}

export function listInspectors(): Inspector[] {
  return INSPECTORS
}

export function allOrderIds(): string[] {
  return ORDERS.map((o) => o.id)
}

export type AttentionCounts = {
  penugasan: number
  pembatalan: number
  basi: number
  terlambat: number
}

export function attentionCounts(): AttentionCounts {
  return {
    penugasan: ORDERS.filter(needsAssignment).length,
    pembatalan: ORDERS.filter((o) => o.cancellationRequested).length,
    basi: ORDERS.filter((o) => isStale(o, DEMO_NOW)).length,
    terlambat: ORDERS.filter(hasLateRejected).length,
  }
}

export function statusDistribution(): Array<{ status: Status; count: number }> {
  const order: Status[] = [
    "requested",
    "assigned",
    "on_the_way",
    "on_site",
    "in_progress",
    "completed",
    "failed",
    "cancelled",
  ]
  return order
    .map((status) => ({ status, count: ORDERS.filter((o) => o.status === status).length }))
    .filter((s) => s.count > 0)
}
