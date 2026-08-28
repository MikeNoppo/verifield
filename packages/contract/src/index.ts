/**
 * Kontrak API Verifield.
 *
 * Sumber kebenarannya adalah anotasi Swagger pada handler Go. Berkas
 * `schema.ts` di sebelah ini di-generate, tidak ditulis tangan:
 *
 *   anotasi Go → swag → swagger.json → swagger2openapi → openapi.json
 *                                     → openapi-typescript → schema.ts
 *
 * Yang ditulis tangan hanyalah berkas ini: nama yang ramah dibaca, dan
 * penandaan field mana yang benar-benar selalu ada. Swagger 2.0 tidak
 * membedakan field wajib pada response, sehingga seluruhnya keluar sebagai
 * opsional — padahal hampir semuanya selalu terisi.
 *
 * `bun run contract:check` memastikan schema.ts yang ter-commit masih sesuai
 * dengan anotasi Go. Tanpa pemeriksaan itu, tipe ini hanya menyalin keadaan
 * pada suatu saat, bukan menjaganya tetap sama.
 */
import type { components } from "./schema"

type Schemas = components["schemas"]

/** Menandai seluruh field wajib, kecuali yang memang bisa tidak ada. */
type WajibKecuali<T, K extends keyof T> = Required<Omit<T, K>> & Pick<T, K>

// ---------------------------------------------------------------------------
// Nilai berbatas
// ---------------------------------------------------------------------------

export type Status = NonNullable<
  Schemas["verifield-be_internal_modules_joborder_dto.JobOrderResponse"]["current_status"]
>

export type Role = NonNullable<
  Schemas["verifield-be_internal_modules_joborder_dto.JobStatusEventResponse"]["actor_role"]
>

export type RejectionReason = NonNullable<
  Schemas["verifield-be_internal_modules_joborder_dto.JobStatusEventResponse"]["rejection_reason"]
>

export type CancelFee = NonNullable<
  Schemas["verifield-be_internal_modules_joborder_dto.CancelResult"]["fee"]
>

export type CancelOutcome = NonNullable<
  Schemas["verifield-be_internal_modules_joborder_dto.CancelResult"]["status"]
>

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

export type FieldError = Required<Schemas["verifield-be_internal_common_apperror.FieldError"]>

export type Meta = Required<Schemas["verifield-be_internal_common_response.Meta"]>

/** Riwayat hanya ikut pada endpoint detail; permintaan pembatalan hanya ada
    ketika benar-benar ada yang menunggu keputusan.

    Kedua field bersarang ditimpa eksplisit — lihat catatan tentang Required<>
    di bawah. */
export type JobOrderDTO = Required<
  Omit<
    Schemas["verifield-be_internal_modules_joborder_dto.JobOrderResponse"],
    "events" | "pending_cancellation"
  >
> & {
  events?: StatusEventDTO[]
  pending_cancellation?: PendingCancellationDTO
}

/** Nilai kosong berarti event pertama (tidak punya status asal), event buatan
    sistem (tidak punya aktor), event yang diterima (tidak punya alasan
    penolakan), atau event tanpa penanda perangkat. */
export type StatusEventDTO = WajibKecuali<
  Schemas["verifield-be_internal_modules_joborder_dto.JobStatusEventResponse"],
  "from_status" | "actor_id" | "rejection_reason" | "reason" | "client_event_id"
>

export type PendingCancellationDTO = Required<
  Schemas["verifield-be_internal_modules_joborder_dto.PendingCancellation"]
>

/** WARNING: Required<> hanya menyentuh lapisan terluar. Field bersarang tetap
    memakai bentuk mentah hasil generate, di mana semuanya opsional — sehingga
    setiap field yang isinya tipe lain di berkas ini harus ditimpa eksplisit.
    Tanpa itu, order di dalam hasil mutasi bertipe berbeda dari order hasil
    endpoint baca, padahal isinya sama persis. */
export type SubmitEventResultDTO = Omit<
  Required<Schemas["verifield-be_internal_modules_joborder_dto.SubmitStatusEventResult"]>,
  "event" | "order"
> & { event: StatusEventDTO; order: JobOrderDTO }

export type CancelResultDTO = Omit<
  Required<Schemas["verifield-be_internal_modules_joborder_dto.CancelResult"]>,
  "order"
> & { order: JobOrderDTO }

export type InspectorDTO = Required<Schemas["internal_modules_reference.InspectorResponse"]>

export type InspectionTypeDTO = Required<
  Schemas["internal_modules_reference.InspectionTypeResponse"]
>

/** Staf internal tidak terikat perusahaan mana pun, sehingga kedua kolom
    perusahaan bisa kosong. */
export type ActorDTO = WajibKecuali<
  Schemas["internal_modules_reference.DemoActorResponse"],
  "company_id" | "company_name"
>

// ---------------------------------------------------------------------------
// Request
// ---------------------------------------------------------------------------

export type CreateJobOrderDTO = Required<
  Schemas["verifield-be_internal_modules_joborder_dto.CreateJobOrderDTO"]
>

export type AssignInspectorDTO = Required<
  Schemas["verifield-be_internal_modules_joborder_dto.AssignInspectorDTO"]
>

export type SubmitStatusEventDTO = WajibKecuali<
  Schemas["verifield-be_internal_modules_joborder_dto.SubmitStatusEventDTO"],
  "occurred_at" | "reason"
>

export type CancelJobOrderDTO = WajibKecuali<
  Schemas["verifield-be_internal_modules_joborder_dto.CancelJobOrderDTO"],
  "expected_version"
>

export type DecideCancellationDTO = WajibKecuali<
  Schemas["verifield-be_internal_modules_joborder_dto.DecideCancellationDTO"],
  "note"
>

export type CorrectStatusDTO = Required<
  Schemas["verifield-be_internal_modules_joborder_dto.CorrectStatusDTO"]
>

export type { components, paths } from "./schema"
