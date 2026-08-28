import { OpsOrderList } from "@/components/verifield/ops-order-list"
import { listInspectors, listOrders } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import { first } from "@/lib/actor/link"
import type { AttentionKey } from "@/lib/domain/summary"

const ATTENTION_KEYS = ["unassigned", "cancellation", "stale", "late_update"] as const

export default async function OpsDashboard({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const sp = await searchParams
  const a = first(sp.a)
  const attention = ATTENTION_KEYS.find((k) => k === a) as AttentionKey | undefined

  const actor = await actorFor("ops", first(sp.actor))
  // Seluruh order diambil sekali, lalu saringan dan penghitungan dilakukan di
  // klien supaya semuanya bergerak bersama saat pembaruan masuk. Bergantung pada
  // asumsi A-06: order aktif berjumlah puluhan, bukan puluhan ribu.
  const [{ orders }, inspectors] = await Promise.all([
    listOrders(actor.id, { limit: 100, sort_by: "status_changed_at" }),
    listInspectors(),
  ])

  return <OpsOrderList orders={orders} inspectors={inspectors} attention={attention} />
}
