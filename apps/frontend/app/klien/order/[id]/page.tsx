import { notFound } from "next/navigation"

import { KlienOrderDetail } from "@/components/verifield/klien-order-detail"
import { ApiError } from "@/lib/api/client"
import { getOrder } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import type { JobOrder } from "@/lib/domain/types"

export default async function KlienOrderDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const actor = await actorFor("klien")

  let order: JobOrder
  try {
    // Kerahasiaan komersial antar klien: order milik perusahaan lain dijawab
    // tidak ditemukan oleh backend, bukan tidak boleh dibuka (asumsi A-03).
    order = await getOrder(actor.id, id)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound()
    throw error
  }

  return <KlienOrderDetail order={order} />
}
