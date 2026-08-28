import { describe, expect, test } from "bun:test"

import { first, withActor } from "./link"

describe("first", () => {
  test("nilai berulang memakai yang pertama", () => {
    expect(first(["a", "b"])).toBe("a")
  })

  test("kosong dan tidak ada sama-sama string kosong", () => {
    expect(first(undefined)).toBe("")
    expect(first([])).toBe("")
  })
})

describe("withActor", () => {
  test("aktor disisipkan ke href polos", () => {
    expect(String(withActor("/klien/order/abc", "budi"))).toBe("/klien/order/abc?actor=budi")
  })

  test("query yang sudah ada dipertahankan", () => {
    expect(String(withActor("/klien?f=berjalan", "budi"))).toBe("/klien?f=berjalan&actor=budi")
  })

  test("aktor lama ditimpa, tidak digandakan", () => {
    expect(String(withActor("/klien?actor=lama", "budi"))).toBe("/klien?actor=budi")
  })

  test("fragmen dipertahankan", () => {
    expect(String(withActor("/klien/order/abc#riwayat", "budi"))).toBe(
      "/klien/order/abc?actor=budi#riwayat",
    )
  })

  test("tanpa aktor href tidak disentuh", () => {
    expect(String(withActor("/klien?f=berjalan", null))).toBe("/klien?f=berjalan")
  })

  // Merakit ulang dari pathname akan menghapus origin dan mengubah tautan luar
  // menjadi tautan internal yang salah.
  test("tujuan di luar aplikasi dibiarkan utuh", () => {
    expect(String(withActor("https://contoh.id/x", "budi"))).toBe("https://contoh.id/x")
    expect(String(withActor("mailto:a@contoh.id", "budi"))).toBe("mailto:a@contoh.id")
    expect(String(withActor("//contoh.id/x", "budi"))).toBe("//contoh.id/x")
  })
})
