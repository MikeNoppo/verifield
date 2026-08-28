import { OpsOrderList } from "@/components/verifield/ops-order-list"
import { listInspectors, listOrders } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import type { AttentionKey } from "@/lib/domain/summary"

const ATTENTION_KEYS = ["penugasan", "pembatalan", "basi", "terlambat"] as const

function first(v: string | string[] | undefined): string {
  return Array.isArray(v) ? (v[0] ?? "") : (v ?? "")
}

export default async function OpsDashboard({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const a = first((await searchParams).a)
  const attention = ATTENTION_KEYS.find((k) => k === a) as AttentionKey | undefined

  const actor = await actorFor("ops")
  // Seluruh order diambil sekali, lalu saringan dan penghitungan dilakukan di
  // klien supaya semuanya bergerak bersama saat pembaruan masuk. Bergantung pada
  // asumsi A-06: order aktif berjumlah puluhan, bukan puluhan ribu.
  const [{ orders }, inspectors] = await Promise.all([
    listOrders(actor.id, { limit: 100, sort_by: "status_changed_at" }),
    listInspectors(),
  ])

  return <OpsOrderList orders={orders} inspectors={inspectors} attention={attention} />
}
