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
  company_id?: string
  inspector_id?: string
  attention?: string
}

/** Alamat backend dilihat dari browser. Server Component memakai nilai yang sama
    karena keduanya berjalan di jaringan yang sama pada PoC ini. */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1"

/** Header identitas pengganti autentikasi. Backend memuatnya menjadi user
    sungguhan, sehingga peran dan kepemilikan tetap ditegakkan di server. */
export const ACTOR_HEADER = "X-Actor-Id"

export function toQueryString(q: ListQuery): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== "") p.set(k, String(v))
  }
  const s = p.toString()
  return s ? `?${s}` : ""
}

/** Error yang membawa pesan siap tampil dari backend.
 *
 *  Pesan backend dipakai apa adanya karena di sanalah kalimat penolakan yang
 *  bisa dimengerti orang non-teknis disusun (F-05) — menuliskannya ulang di sini
 *  berarti dua tempat yang harus dijaga tetap sama. */
export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly fields: FieldError[]

  constructor(message: string, code: string, status: number, fields: FieldError[] = []) {
    super(message)
    this.name = "ApiError"
    this.code = code
    this.status = status
    this.fields = fields
  }

  /** Perubahan ditolak karena data sudah berubah lebih dulu (B-09). */
  get isConflict(): boolean {
    return this.status === 409
  }
}

export type FetchOptions = RequestInit & { actorId?: string }

export async function apiFetch<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const { actorId, headers, ...init } = options

  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(actorId ? { [ACTOR_HEADER]: actorId } : {}),
      ...headers,
    },
    // Data order berubah terus-menerus; menyimpannya di cache akan menampilkan
    // status basi persis pada layar yang tujuannya justru menghapus kebasian.
    cache: "no-store",
  })

  if (res.status === 204) return undefined as T

  const body = (await res.json()) as Envelope<T>

  if (!body.success) {
    throw new ApiError(body.message, body.code, res.status, body.errors ?? [])
  }

  return body.data
}

/** Varian yang ikut mengembalikan meta paginasi. */
export async function apiFetchList<T>(
  path: string,
  options: FetchOptions = {},
): Promise<{ data: T; total: number }> {
  const { actorId, headers, ...init } = options

  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(actorId ? { [ACTOR_HEADER]: actorId } : {}),
      ...headers,
    },
    cache: "no-store",
  })

  const body = (await res.json()) as Envelope<T>

  if (!body.success) {
    throw new ApiError(body.message, body.code, res.status, body.errors ?? [])
  }

  return { data: body.data, total: body.meta?.total ?? 0 }
}
