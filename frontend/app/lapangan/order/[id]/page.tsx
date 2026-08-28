import { notFound } from "next/navigation"

import { LapanganOrderDetail } from "@/components/verifield/lapangan-order-detail"
import { ApiError } from "@/lib/api/client"
import { getOrder } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import type { JobOrder } from "@/lib/domain/types"

export default async function LayarAksi({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  const actor = await actorFor("lapangan")

  let order: JobOrder
  try {
    order = await getOrder(actor.id, id)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound()
    throw error
  }

  return <LapanganOrderDetail order={order} />
}
