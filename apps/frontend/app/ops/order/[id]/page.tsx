import { notFound } from "next/navigation"

import { OpsOrderDetail } from "@/components/verifield/ops-order-detail"
import { ApiError } from "@/lib/api/client"
import { getOrder, listInspectors } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import type { Inspector, JobOrder } from "@/lib/domain/types"

export default async function OpsOrderDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const actor = await actorFor("ops")

  let order: JobOrder
  let inspectors: Inspector[]
  try {
    ;[order, inspectors] = await Promise.all([getOrder(actor.id, id), listInspectors()])
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound()
    throw error
  }

  return <OpsOrderDetail order={order} inspectors={inspectors} />
}
