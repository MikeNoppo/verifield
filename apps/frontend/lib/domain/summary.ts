import { hasOpenAlert, isStale, isTerminal, needsAssignment } from "./status"
import type { JobOrder, Status } from "./types"

export type AttentionKey = "unassigned" | "cancellation" | "stale" | "late_update"

export type AttentionCounts = Record<AttentionKey, number>

/** Dihitung di klien dari daftar order yang sudah diambil, bukan lewat empat
    permintaan hitung terpisah ke backend.
 *
 *  Ini bergantung pada asumsi A-06: order aktif berjumlah puluhan, bukan puluhan
 *  ribu, sehingga seluruhnya muat dalam satu halaman. Begitu asumsi itu tidak
 *  berlaku lagi, perhitungannya harus pindah ke backend sebagai query agregat —
 *  bukan diperbesar limit-nya. */
export function attentionCounts(orders: JobOrder[], now: Date): AttentionCounts {
  return {
    unassigned: orders.filter(needsAssignment).length,
    cancellation: orders.filter((o) => o.cancellationRequested).length,
    stale: orders.filter((o) => isStale(o, now)).length,
    late_update: orders.filter(hasOpenAlert).length,
  }
}

const DISTRIBUTION_ORDER: Status[] = [
  "requested",
  "assigned",
  "on_the_way",
  "on_site",
  "in_progress",
  "completed",
  "failed",
  "cancelled",
]

export function statusDistribution(
  orders: JobOrder[],
): Array<{ status: Status; count: number }> {
  return DISTRIBUTION_ORDER.map((status) => ({
    status,
    count: orders.filter((o) => o.status === status).length,
  })).filter((s) => s.count > 0)
}

export function countRunning(orders: JobOrder[]): number {
  return orders.filter((o) => !isTerminal(o.status)).length
}

export type Bucket = "berjalan" | "selesai" | "bermasalah"

/** Saringan layar klien. Sengaja tidak dipetakan ke parameter status backend:
    "bermasalah" menggabungkan tiga kondisi yang berbeda, dan menerjemahkannya
    jadi rangkaian permintaan terpisah hanya menambah bolak-balik tanpa manfaat
    pada skala ini. */
export function inBucket(order: JobOrder, bucket: Bucket | undefined): boolean {
  if (!bucket) return true
  if (bucket === "berjalan") return !isTerminal(order.status)
  if (bucket === "selesai") return order.status === "completed"
  return order.status === "failed" || order.status === "cancelled" || hasOpenAlert(order)
}
