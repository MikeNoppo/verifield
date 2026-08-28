import { afterEach, describe, expect, test } from "bun:test"

import { apply, getSnapshot, reset } from "./store"
import type { JobOrder } from "@/lib/domain/types"

function order(id: string, seq: number, status: JobOrder["status"] = "on_site"): JobOrder {
  return {
    id,
    ref: `JO-2026-${id}`,
    clientName: "PT Contoh",
    inspectionType: "Inspeksi Kargo Curah",
    object: "Kargo curah",
    location: "Dermaga 3",
    address: "Jl. Contoh",
    city: "Jakarta",
    scheduledAt: "2026-08-28T00:00:00Z",
    scheduledEndAt: "2026-08-28T06:00:00Z",
    inspectorId: null,
    inspectorName: null,
    status,
    statusChangedAt: "2026-08-28T00:00:00Z",
    exitStatus: null,
    version: 1,
    seq,
    cancellationRequested: false,
    hasOpenAlert: false,
    pendingCancellation: null,
    events: [],
    createdAt: "2026-08-28T00:00:00Z",
    updatedAt: "2026-08-28T00:00:00Z",
  }
}

afterEach(() => reset())

describe("penggabungan berdasarkan seq", () => {
  test("pesan yang lebih baru diterapkan", () => {
    expect(apply(order("a", 5))).toBe(true)
    expect(apply(order("a", 6, "in_progress"))).toBe(true)
    expect(getSnapshot().orders.get("a")?.status).toBe("in_progress")
  })

  test("pesan ganda diabaikan", () => {
    // Replay saat menyambung ulang sengaja tumpang tindih dengan siaran
    // langsung agar tidak ada celah, sehingga pesan yang sama memang bisa
    // datang dua kali.
    apply(order("a", 5))
    expect(apply(order("a", 5))).toBe(false)
  })

  test("pesan yang datang terbalik urutannya tidak membuat status mundur", () => {
    apply(order("a", 9, "completed"))
    expect(apply(order("a", 7, "on_site"))).toBe(false)
    expect(getSnapshot().orders.get("a")?.status).toBe("completed")
  })

  test("kursor mengikuti seq tertinggi yang pernah diterima", () => {
    apply(order("a", 4))
    apply(order("b", 11))
    apply(order("a", 6))
    expect(getSnapshot().cursor).toBe(11)
  })

  test("order berbeda tidak saling menimpa", () => {
    apply(order("a", 4, "on_site"))
    apply(order("b", 5, "completed"))
    expect(getSnapshot().orders.get("a")?.status).toBe("on_site")
    expect(getSnapshot().orders.get("b")?.status).toBe("completed")
  })

  test("waktu kabar terakhir ikut tercatat", () => {
    expect(getSnapshot().lastMessageAt).toBeNull()
    apply(order("a", 1))
    expect(getSnapshot().lastMessageAt).not.toBeNull()
  })
})
