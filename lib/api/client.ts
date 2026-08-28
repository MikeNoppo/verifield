import type { FieldError } from "./dto"

/** Bentuk baku seluruh response Verifield BE (Go + Gin), disalin apa adanya dari
    internal/common/response/response.go. Dibuat discriminated union supaya
    pemanggil tidak perlu menjaga sendiri bahwa data ada ketika success. */
export type Envelope<T> =
  | {
      success: true
      message: string
      data: T
      meta?: { page: number; limit: number; total: number; total_page: number }
    }
  | {
      success: false
      message: string
      code: string
      errors?: FieldError[]
    }

export type ListQuery = {
  page?: number
  limit?: number
  search?: string
  sort_by?: string
  sort_dir?: "asc" | "desc"
  status?: string
  assigned_to?: string
}

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1"

export function toQueryString(q: ListQuery): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== "") p.set(k, String(v))
  }
  const s = p.toString()
  return s ? `?${s}` : ""
}

/** Belum dipakai selama data masih mock, tetapi sengaja ditulis sekarang agar
    bentuk pemanggilan di lib/api/orders.ts sudah final. */
export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<Envelope<T>> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  })
  return (await res.json()) as Envelope<T>
}
