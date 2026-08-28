import { describe, expect, test } from "bun:test"

import { findActor, identityLabel, resolveActor } from "./resolve"
import type { Actor } from "@/lib/domain/types"

function actor(id: string, role: Actor["role"], companyName: string | null = null): Actor {
  return {
    id,
    name: `Nama ${id}`,
    email: `${id}@contoh.id`,
    role,
    companyId: companyName ? `co-${id}` : null,
    companyName,
  }
}

const ACTORS: Actor[] = [
  actor("admin", "admin"),
  actor("budi", "client", "PT Samudra"),
  actor("dewi", "client", "PT Bara"),
  actor("joko", "inspector"),
  actor("rina", "inspector"),
]

describe("findActor", () => {
  test("id yang cocok peran dipilih", () => {
    expect(findActor(ACTORS, "klien", "dewi")?.id).toBe("dewi")
  })

  test("tanpa id jatuh ke aktor pertama berperan itu", () => {
    expect(findActor(ACTORS, "klien", null)?.id).toBe("budi")
    expect(findActor(ACTORS, "lapangan", null)?.id).toBe("joko")
  })

  // Aktor peran lain di URL tidak boleh diterima: layar klien yang menampilkan
  // inspektor akan meminta order yang tidak boleh dilihatnya.
  test("id milik peran lain diabaikan, bukan dipakai", () => {
    expect(findActor(ACTORS, "klien", "joko")?.id).toBe("budi")
  })

  test("id yang tidak dikenal diabaikan", () => {
    expect(findActor(ACTORS, "ops", "entah")?.id).toBe("admin")
  })

  test("peran tanpa aktor mengembalikan undefined, tidak melempar", () => {
    expect(findActor([actor("admin", "admin")], "lapangan", null)).toBeUndefined()
  })
})

describe("resolveActor", () => {
  test("melempar saat tidak ada aktor berperan itu", () => {
    expect(() => resolveActor([actor("admin", "admin")], "klien", null)).toThrow(/client/)
  })
})

describe("identityLabel", () => {
  test("klien memakai nama perusahaan", () => {
    expect(identityLabel("klien", actor("budi", "client", "PT Samudra"))).toBe("PT Samudra")
  })

  test("klien tanpa perusahaan jatuh ke namanya", () => {
    expect(identityLabel("klien", actor("budi", "client"))).toBe("Nama budi")
  })

  test("koordinator dan inspektor memakai namanya", () => {
    expect(identityLabel("ops", actor("admin", "admin"))).toBe("Nama admin · Koordinator Operasional")
    expect(identityLabel("lapangan", actor("joko", "inspector"))).toBe("Nama joko")
  })
})
