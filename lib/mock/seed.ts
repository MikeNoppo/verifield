import type { EventKind, Inspector, JobOrder, Role, Status, StatusEvent } from "@/lib/domain/types"
import { DEMO_NOW } from "@/lib/demo-time"

/** Seluruh data dihitung relatif terhadap satu titik waktu tetap. Tanpa ini,
    skenario "tanpa pembaruan 9 jam" akan berhenti benar keesokan harinya. */
export { DEMO_NOW } from "@/lib/demo-time"

/** Inspektor yang sedang memakai layar /lapangan. Tidak ada autentikasi di PoC. */
export const CURRENT_INSPECTOR_ID = "ins-a07"

/** Klien yang sedang memakai layar /klien. Klien hanya melihat order miliknya (A-03). */
export const CURRENT_CLIENT = "PT Karya Bahari Ekspor"

function iso(hAgo: number, minAgo = 0): string {
  return new Date(DEMO_NOW.getTime() - hAgo * 3_600_000 - minAgo * 60_000).toISOString()
}

export const INSPECTORS: Inspector[] = [
  { id: "ins-a07", name: "Adi Nugroho", code: "INS-A07", city: "Jakarta", activeJobs: 4 },
  { id: "ins-b12", name: "Budi Santoso", code: "INS-B12", city: "Jakarta", activeJobs: 3 },
  { id: "ins-c03", name: "Citra Wulandari", code: "INS-C03", city: "Surabaya", activeJobs: 2 },
  { id: "ins-d18", name: "Dedi Kurniawan", code: "INS-D18", city: "Balikpapan", activeJobs: 3 },
  { id: "ins-e05", name: "Eka Prasetyo", code: "INS-E05", city: "Bontang", activeJobs: 2 },
  { id: "ins-f21", name: "Fitri Handayani", code: "INS-F21", city: "Cilacap", activeJobs: 1 },
  { id: "ins-g09", name: "Gunawan Halim", code: "INS-G09", city: "Samarinda", activeJobs: 2 },
  { id: "ins-h14", name: "Hendra Wijaya", code: "INS-H14", city: "Surabaya", activeJobs: 1 },
  { id: "ins-i06", name: "Indah Permata", code: "INS-I06", city: "Jakarta", activeJobs: 2 },
  { id: "ins-j11", name: "Joko Susilo", code: "INS-J11", city: "Balikpapan", activeJobs: 0 },
]

type Step = {
  to: Status
  /** Jam sebelum DEMO_NOW saat tombol benar-benar ditekan di lapangan. */
  hAgo: number
  minAgo?: number
  /** Jeda menit antara kejadian dan saat sistem menerimanya. Nol berarti online. */
  lagMin?: number
  kind?: EventKind
  reason?: string
  actor?: { role: Role; name: string }
}

type Spec = {
  ref: string
  clientName: string
  inspectionType: string
  object: string
  location: string
  city: string
  schedHAgo: number
  inspector?: string
  steps: Step[]
  cancellationRequested?: boolean
}

function build(spec: Spec): JobOrder {
  const id = spec.ref.toLowerCase()
  const ins = INSPECTORS.find((i) => i.id === spec.inspector)

  let prev: Status | null = null
  const events: StatusEvent[] = spec.steps.map((s, i) => {
    const kind = s.kind ?? "transition"
    const eventTime = iso(s.hAgo, s.minAgo ?? 0)
    const lag = s.lagMin ?? 0
    const actor =
      s.actor ??
      (i === 0
        ? { role: "klien" as Role, name: spec.clientName }
        : s.to === "assigned"
          ? { role: "ops" as Role, name: "Rina Oktaviani" }
          : { role: "lapangan" as Role, name: ins?.name ?? "—" })

    const ev: StatusEvent = {
      id: `${id}-e${i + 1}`,
      orderId: id,
      kind,
      from: prev,
      to: s.to,
      eventTime,
      receivedTime: new Date(new Date(eventTime).getTime() + lag * 60_000).toISOString(),
      actorRole: actor.role,
      actorName: actor.name,
      reason: s.reason,
      idempotencyKey: `${id}-${i + 1}-8f3a2c`,
    }
    // Kejadian yang ditolak tidak pernah menggeser status, jadi tidak boleh
    // menjadi titik asal transisi berikutnya (B-07).
    if (kind === "transition") prev = s.to
    return ev
  })

  const transitions = events.filter((e) => e.kind === "transition")
  const status = transitions[transitions.length - 1]!.to
  const lastEvent = events[events.length - 1]!

  return {
    id,
    ref: spec.ref,
    clientName: spec.clientName,
    inspectionType: spec.inspectionType,
    object: spec.object,
    location: spec.location,
    city: spec.city,
    scheduledAt: iso(spec.schedHAgo),
    inspectorId: ins?.id ?? null,
    inspectorName: ins?.name ?? null,
    status,
    version: events.length + 1,
    cancellationRequested: spec.cancellationRequested ?? false,
    events,
    createdAt: events[0]!.eventTime,
    updatedAt: lastEvent.receivedTime,
  }
}

const SCENARIOS: Spec[] = [
  // Sinyal hilang tiga jam, tiga kejadian terkirim sekaligus pukul 11.40.
  // Urutan riwayat mengikuti waktu kejadian, bukan waktu terima (B-02, B-06).
  {
    ref: "JO-2601",
    clientName: CURRENT_CLIENT,
    inspectionType: "Sampling Batu Bara",
    object: "Stockpile batu bara Blok C, 12.000 MT",
    location: "Terminal Batubara Bontang",
    city: "Bontang",
    schedHAgo: 6,
    inspector: "ins-e05",
    steps: [
      { to: "requested", hAgo: 30 },
      { to: "assigned", hAgo: 28 },
      { to: "on_the_way", hAgo: 7, minAgo: 30 },
      { to: "on_site", hAgo: 5, minAgo: 16, lagMin: 146 },
      { to: "in_progress", hAgo: 5, minAgo: 10, lagMin: 140 },
      { to: "completed", hAgo: 3, minAgo: 25, lagMin: 35 },
    ],
  },
  // Klien membatalkan saat inspektor offline. Inspektor tetap menyelesaikan
  // pekerjaan; laporannya masuk setelah order final, ditolak tetapi dicatat (B-07).
  {
    ref: "JO-2602",
    clientName: CURRENT_CLIENT,
    inspectionType: "Draft Survey",
    object: "Kargo curah 8.500 MT",
    location: "Pelabuhan Tanjung Priok",
    city: "Jakarta",
    schedHAgo: 8,
    inspector: "ins-a07",
    steps: [
      { to: "requested", hAgo: 26 },
      { to: "assigned", hAgo: 24 },
      { to: "on_the_way", hAgo: 9 },
      { to: "on_site", hAgo: 7, minAgo: 40 },
      { to: "in_progress", hAgo: 7 },
      {
        to: "cancelled",
        hAgo: 4,
        minAgo: 30,
        reason: "Pembeli menarik diri dari transaksi, disetujui koordinator.",
        actor: { role: "ops", name: "Rina Oktaviani" },
      },
      {
        to: "completed",
        hAgo: 3,
        lagMin: 90,
        kind: "late_rejected",
        reason:
          "Inspektor berada di luar jangkauan sinyal saat pembatalan diputuskan dan menyelesaikan pemeriksaan. Kompensasi kunjungan perlu diselesaikan koordinator.",
      },
    ],
  },
  // Kargo belum tiba di dermaga. Bukan pembatalan — perusahaan tetap berhak
  // menagih biaya kunjungan, dan angka itu hanya terukur bila Failed terpisah.
  {
    ref: "JO-2603",
    clientName: "PT Nusantara Coal Trading",
    inspectionType: "Draft Survey",
    object: "Kargo curah 6.200 MT",
    location: "Pelabuhan Tanjung Intan",
    city: "Cilacap",
    schedHAgo: 5,
    inspector: "ins-f21",
    steps: [
      { to: "requested", hAgo: 22 },
      { to: "assigned", hAgo: 20 },
      { to: "on_the_way", hAgo: 6 },
      { to: "on_site", hAgo: 4, minAgo: 20 },
      {
        to: "failed",
        hAgo: 3,
        minAgo: 55,
        reason: "Kargo belum tiba di dermaga. Kapal tertahan menunggu slot sandar.",
      },
    ],
  },
  // Tidak ada pembaruan sejak sembilan jam lalu pada hari kerja. Koordinator
  // ditandai agar bisa menindaklanjuti sebelum klien menelepon.
  {
    ref: "JO-2604",
    clientName: "PT Borneo Mineral Utama",
    inspectionType: "Verifikasi Stockpile",
    object: "Tumpukan nikel ore Blok A",
    location: "Stockpile Palaran",
    city: "Samarinda",
    schedHAgo: 11,
    inspector: "ins-g09",
    steps: [
      { to: "requested", hAgo: 32 },
      { to: "assigned", hAgo: 30 },
      { to: "on_the_way", hAgo: 9, minAgo: 40 },
    ],
  },
  // Pembatalan diajukan setelah pekerjaan dimulai. Bukan tindakan langsung —
  // menunggu keputusan koordinator (B-05).
  {
    ref: "JO-2605",
    clientName: CURRENT_CLIENT,
    inspectionType: "Kalibrasi Tangki",
    object: "Tangki penyimpanan T-04",
    location: "Terminal Kariangau",
    city: "Balikpapan",
    schedHAgo: 4,
    inspector: "ins-d18",
    cancellationRequested: true,
    steps: [
      { to: "requested", hAgo: 27 },
      { to: "assigned", hAgo: 25 },
      { to: "on_the_way", hAgo: 5, minAgo: 30 },
      { to: "on_site", hAgo: 3, minAgo: 50 },
      { to: "in_progress", hAgo: 3, minAgo: 10 },
      {
        to: "in_progress",
        hAgo: 0,
        minAgo: 45,
        kind: "cancellation_request",
        reason: "Jadwal pengapalan dimajukan, klien meminta pemeriksaan dihentikan.",
        actor: { role: "klien", name: CURRENT_CLIENT },
      },
    ],
  },
  // Inspektor salah menekan "Selesai" padahal baru tiba. Dikoreksi koordinator
  // beralasan, sebagai entri baru — entri lama tetap utuh (B-06, F-06).
  {
    ref: "JO-2606",
    clientName: "PT Agrindo Sejahtera",
    inspectionType: "Analisa Kargo Curah",
    object: "Kargo CPO 3.200 MT",
    location: "Pelabuhan Tanjung Perak",
    city: "Surabaya",
    schedHAgo: 6,
    inspector: "ins-c03",
    steps: [
      { to: "requested", hAgo: 21 },
      { to: "assigned", hAgo: 19 },
      { to: "on_the_way", hAgo: 6, minAgo: 20 },
      { to: "on_site", hAgo: 4, minAgo: 35 },
      { to: "completed", hAgo: 4, minAgo: 30 },
      {
        to: "on_site",
        hAgo: 4,
        minAgo: 5,
        kind: "correction",
        reason:
          "Inspektor menghubungi koordinator: tombol Selesai tertekan tidak sengaja saat baru tiba. Status dikembalikan ke Di Lokasi.",
        actor: { role: "ops", name: "Rina Oktaviani" },
      },
      { to: "in_progress", hAgo: 3, minAgo: 40 },
    ],
  },
]

const CLIENTS = [
  CURRENT_CLIENT,
  "PT Nusantara Coal Trading",
  "PT Agrindo Sejahtera",
  "PT Delta Petro Kimia",
  "PT Samudera Logistik Prima",
  "PT Borneo Mineral Utama",
  "PT Cahaya Tani Nusantara",
  "PT Meridian Commodity",
]

const SITES: Array<[string, string]> = [
  ["Pelabuhan Tanjung Priok", "Jakarta"],
  ["Terminal Batubara Bontang", "Bontang"],
  ["Pelabuhan Tanjung Intan", "Cilacap"],
  ["Terminal Kariangau", "Balikpapan"],
  ["Pelabuhan Tanjung Perak", "Surabaya"],
  ["Stockpile Palaran", "Samarinda"],
]

const TYPES = [
  "Draft Survey",
  "Sampling Batu Bara",
  "Kalibrasi Tangki",
  "Pemeriksaan Kontainer",
  "Analisa Kargo Curah",
  "Verifikasi Stockpile",
]

const OBJECTS = [
  "Kargo curah 8.500 MT",
  "Stockpile batu bara Blok B",
  "Tangki penyimpanan T-11",
  "12 kontainer 40ft",
  "Kargo CPO 3.200 MT",
  "Tumpukan nikel ore Blok D",
  "Kargo curah 4.750 MT",
  "Tangki penyimpanan T-02",
]

/** Jalur yang dilalui order rutin, ditulis sebagai jam-sebelum-sekarang untuk
    tiap tahap. Cukup untuk mengisi tabel koordinator dengan kepadatan nyata. */
const ROUTES: Array<{ steps: Array<[Status, number]>; ins: string }> = [
  { steps: [["requested", 3]], ins: "" },
  { steps: [["requested", 2]], ins: "" },
  { steps: [["requested", 1]], ins: "" },
  { steps: [["requested", 8], ["assigned", 5]], ins: "ins-b12" },
  { steps: [["requested", 6], ["assigned", 4]], ins: "ins-i06" },
  { steps: [["requested", 14], ["assigned", 11]], ins: "ins-h14" },
  { steps: [["requested", 28], ["assigned", 26], ["on_the_way", 2]], ins: "ins-a07" },
  { steps: [["requested", 25], ["assigned", 23], ["on_the_way", 1]], ins: "ins-b12" },
  { steps: [["requested", 27], ["assigned", 25], ["on_the_way", 3], ["on_site", 1]], ins: "ins-a07" },
  { steps: [["requested", 24], ["assigned", 22], ["on_the_way", 4], ["on_site", 2]], ins: "ins-j11" },
  {
    steps: [["requested", 29], ["assigned", 27], ["on_the_way", 5], ["on_site", 3], ["in_progress", 2]],
    ins: "ins-a07",
  },
  {
    steps: [["requested", 26], ["assigned", 24], ["on_the_way", 6], ["on_site", 4], ["in_progress", 3]],
    ins: "ins-c03",
  },
  {
    steps: [["requested", 31], ["assigned", 29], ["on_the_way", 7], ["on_site", 5], ["in_progress", 4]],
    ins: "ins-e05",
  },
  {
    steps: [
      ["requested", 34],
      ["assigned", 32],
      ["on_the_way", 12],
      ["on_site", 10],
      ["in_progress", 9],
      ["completed", 7],
    ],
    ins: "ins-d18",
  },
  {
    steps: [
      ["requested", 40],
      ["assigned", 38],
      ["on_the_way", 15],
      ["on_site", 13],
      ["in_progress", 12],
      ["completed", 10],
    ],
    ins: "ins-f21",
  },
  {
    steps: [
      ["requested", 46],
      ["assigned", 44],
      ["on_the_way", 20],
      ["on_site", 18],
      ["in_progress", 17],
      ["completed", 15],
    ],
    ins: "ins-g09",
  },
  {
    steps: [
      ["requested", 52],
      ["assigned", 50],
      ["on_the_way", 26],
      ["on_site", 24],
      ["in_progress", 23],
      ["completed", 21],
    ],
    ins: "ins-h14",
  },
  {
    steps: [
      ["requested", 58],
      ["assigned", 56],
      ["on_the_way", 32],
      ["on_site", 30],
      ["in_progress", 29],
      ["completed", 27],
    ],
    ins: "ins-i06",
  },
  { steps: [["requested", 36], ["assigned", 34], ["cancelled", 33]], ins: "ins-b12" },
  { steps: [["requested", 44], ["cancelled", 43]], ins: "" },
]

const ROUTINE: Spec[] = Array.from({ length: 39 }, (_, i) => {
  const route = ROUTES[i % ROUTES.length]!
  const [location, city] = SITES[i % SITES.length]!
  const last = route.steps[route.steps.length - 1]!
  return {
    ref: `JO-${2610 + i}`,
    clientName: CLIENTS[i % CLIENTS.length]!,
    inspectionType: TYPES[i % TYPES.length]!,
    object: OBJECTS[i % OBJECTS.length]!,
    location,
    city,
    schedHAgo: last[1] + (i % 3) + (i % 5),
    inspector: route.ins || undefined,
    steps: route.steps.map(([to, hAgo]) => ({ to, hAgo: hAgo + (i % 3) })),
  }
})

export const ORDERS: JobOrder[] = [...SCENARIOS, ...ROUTINE].map(build)
