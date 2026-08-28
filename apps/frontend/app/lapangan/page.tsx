import { LapanganTaskList } from "@/components/verifield/lapangan-task-list"
import { listOrders } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import { first } from "@/lib/actor/link"

export default async function TugasHariIni({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const actor = await actorFor("lapangan", first((await searchParams).actor))
  // Backend memaksakan saringan inspektor sendiri untuk peran inspektor, apa
  // pun yang dikirim di query — batas baca ditegakkan di server.
  const { orders } = await listOrders(actor.id, {
    limit: 100,
    sort_by: "scheduled_start_at",
    sort_dir: "asc",
  })

  return <LapanganTaskList orders={orders} />
}
