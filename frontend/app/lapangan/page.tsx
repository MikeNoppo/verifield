import { LapanganTaskList } from "@/components/verifield/lapangan-task-list"
import { listOrders } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"

export default async function TugasHariIni() {
  const actor = await actorFor("lapangan")
  const { orders } = await listOrders(actor.id, {
    inspector_id: actor.id,
    limit: 100,
    sort_by: "scheduled_start_at",
    sort_dir: "asc",
  })

  return <LapanganTaskList orders={orders} />
}
