import { afterEach, beforeEach, describe, expect, mock, test } from "bun:test"

import type { SubmitEventResult } from "@/lib/api/orders"

// Antrean adalah satu-satunya jalan laporan inspektor menuju server, jadi yang
// diuji di sini adalah perilakunya saat pengiriman GAGAL — bukan saat berhasil.
const submit = mock<
  (actorId: string, id: string, input: { client_event_id: string }) => Promise<SubmitEventResult>
>(() => Promise.resolve(acceptedResult()))

// mock.module mengganti SELURUH modul, sedangkan store real-time ikut memakai
// orderFromDTO dari sana — tanpa stub ini, store gagal dimuat.
mock.module("@/lib/api/orders", () => ({
  submitStatusEvent: submit,
  orderFromDTO: (d: unknown) => d,
}))

const { ApiError } = await import("@/lib/api/client")
const { enqueue, flush, getSnapshot, hydrate, reset } = await import("./queue")

function acceptedResult(accepted = true, message = "Status berhasil diperbarui."): SubmitEventResult {
  return {
    accepted,
    duplicate: false,
    message,
    // Isi lengkapnya tidak relevan untuk perilaku antrean.
    event: {} as SubmitEventResult["event"],
    order: { id: "o1", seq: 1 } as SubmitEventResult["order"],
  }
}

function report(clientEventId: string) {
  return {
    clientEventId,
    orderId: "o1",
    orderRef: "JO-2026-0001",
    toStatus: "on_site" as const,
    occurredAt: "2026-08-28T06:00:00Z",
    actorId: "a1",
  }
}

beforeEach(() => {
  const store = new Map<string, string>()
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    },
  })
  reset()
  submit.mockClear()
})

afterEach(() => reset())

describe("antrean laporan lapangan", () => {
  test("laporan terkirim dikeluarkan dari antrean", async () => {
    submit.mockResolvedValue(acceptedResult())
    enqueue(report("e1"))
    await flush()

    expect(getSnapshot().items).toHaveLength(0)
    expect(submit).toHaveBeenCalledTimes(1)
  })

  test("penanda yang sama tidak pernah masuk antrean dua kali", () => {
    // Pertahanan pertama terhadap tombol yang ditekan berulang kali; server
    // punya pertahanan keduanya lewat unique index (B-03).
    submit.mockImplementation(() => new Promise(() => {}))
    enqueue(report("e1"))
    enqueue(report("e1"))

    expect(getSnapshot().items).toHaveLength(1)
  })

  test("gagal jaringan menahan laporan di antrean", async () => {
    submit.mockRejectedValue(new TypeError("Failed to fetch"))
    enqueue(report("e1"))
    await flush()

    expect(getSnapshot().items).toHaveLength(1)
    expect(getSnapshot().failures).toHaveLength(0)
  })

  test("gagal jaringan menghentikan sisa antrean agar urutannya terjaga", async () => {
    // Melanjutkan ke laporan berikutnya membuat server menerima "Selesai"
    // sebelum "Mulai", yang akan ditolak sebagai keluar urutan padahal sah.
    submit.mockRejectedValue(new TypeError("Failed to fetch"))
    enqueue(report("e1"))
    enqueue(report("e2"))
    await flush()

    expect(getSnapshot().items).toHaveLength(2)
    expect(submit).toHaveBeenCalledTimes(1)
  })

  test("penolakan server mengeluarkan laporan dan menjelaskan alasannya", async () => {
    // Mengulanginya tidak akan berhasil, tetapi menelannya tanpa penjelasan
    // akan membuat inspektor kembali melapor lewat telepon.
    submit.mockRejectedValue(new ApiError("Order ini tidak ditugaskan kepada Anda", "FORBIDDEN", 403))
    enqueue(report("e1"))
    await flush()

    expect(getSnapshot().items).toHaveLength(0)
    expect(getSnapshot().failures).toEqual([
      { orderRef: "JO-2026-0001", message: "Order ini tidak ditugaskan kepada Anda" },
    ])
  })

  test("laporan yang diterima tetapi tidak mengubah status ikut dijelaskan", async () => {
    // Keputusan B-07: pengirimannya berhasil, perubahan statusnya yang ditolak.
    submit.mockResolvedValue(acceptedResult(false, "Order sudah ditutup sebelum laporan Anda masuk."))
    enqueue(report("e1"))
    await flush()

    expect(getSnapshot().items).toHaveLength(0)
    expect(getSnapshot().failures[0]?.message).toContain("sudah ditutup")
  })

  test("laporan yang belum terkirim tersimpan di perangkat", async () => {
    // Inilah yang membuat laporan selamat melewati aplikasi yang ditutup dan
    // halaman yang dimuat ulang.
    submit.mockRejectedValue(new TypeError("Failed to fetch"))
    enqueue(report("e1"))
    await flush()

    const persisted = JSON.parse(localStorage.getItem("verifield.outbox.v1") ?? "[]")
    expect(persisted).toHaveLength(1)
    expect(persisted[0].clientEventId).toBe("e1")
    expect(persisted[0].occurredAt).toBe("2026-08-28T06:00:00Z")
  })

  test("hydrate memulihkan antrean dari penyimpanan perangkat", () => {
    localStorage.setItem(
      "verifield.outbox.v1",
      JSON.stringify([{ ...report("e9"), attempts: 0 }]),
    )
    hydrate()

    expect(getSnapshot().items).toHaveLength(1)
    expect(getSnapshot().items[0]?.clientEventId).toBe("e9")
  })
})
