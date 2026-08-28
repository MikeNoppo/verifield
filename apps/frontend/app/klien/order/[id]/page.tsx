import { notFound } from "next/navigation"

import { KlienOrderDetail } from "@/components/verifield/klien-order-detail"
import { ApiError } from "@/lib/api/client"
import { getOrder } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import { first } from "@/lib/actor/link"
import type { JobOrder } from "@/lib/domain/types"

export default async function KlienOrderDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const { id } = await params
  const actor = await actorFor("klien", first((await searchParams).actor))

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
