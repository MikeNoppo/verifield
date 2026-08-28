import { describe, expect, test } from "bun:test"

import { cancelAuthority, cancelOffered } from "./status"
import type { Role, Status } from "./types"

/** Matriks ini adalah padanan tampilan dari EvaluateCancel di backend. Yang
    dijaga di sini bukan hanya isinya, melainkan urutan pembacaannya: peran lebih
    dulu, baru keadaan. Menukar urutannya membuat inspektor mendengar "ordernya
    sudah selesai" — kalimat yang seolah menjanjikan bahwa pada order lain ia
    akan berhasil. */
describe("cancelAuthority", () => {
  const cases: Array<[string, Role, Status, string]> = [
    ["klien sebelum penugasan", "client", "requested", "allowed"],
    ["klien saat inspektor berangkat", "client", "on_the_way", "allowed"],
    ["klien saat pekerjaan berjalan", "client", "in_progress", "requires_approval"],
    ["koordinator saat pekerjaan berjalan", "admin", "in_progress", "allowed"],
    ["inspektor tidak pernah berwenang", "inspector", "on_site", "forbidden"],
    ["inspektor tetap tidak berwenang pada order final", "inspector", "completed", "forbidden"],
    ["cs hanya membaca", "cs", "requested", "forbidden"],
    ["order selesai", "admin", "completed", "unavailable"],
    ["order sudah dibatalkan", "client", "cancelled", "unavailable"],
  ]

  for (const [name, role, status, kind] of cases) {
    test(name, () => {
      expect(cancelAuthority(role, status).kind).toBe(kind)
    })
  }

  test("setiap penolakan membawa kalimat yang bisa dibaca pengguna", () => {
    for (const [, role, status] of cases) {
      const auth = cancelAuthority(role, status)
      if (auth.kind === "allowed") {
        expect(auth.consequence.length).toBeGreaterThan(0)
      } else {
        expect(auth.message.length).toBeGreaterThan(0)
      }
    }
  })
})

describe("cancelOffered", () => {
  test("hanya menawarkan aksi yang benar-benar bisa dilakukan", () => {
    expect(cancelOffered(cancelAuthority("client", "requested"))).toBe(true)
    expect(cancelOffered(cancelAuthority("client", "in_progress"))).toBe(true)
    expect(cancelOffered(cancelAuthority("inspector", "on_site"))).toBe(false)
    expect(cancelOffered(cancelAuthority("admin", "completed"))).toBe(false)
  })
})
