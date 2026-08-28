import { notFound } from "next/navigation"

import { LapanganOrderDetail } from "@/components/verifield/lapangan-order-detail"
import { ApiError } from "@/lib/api/client"
import { getOrder } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import { first } from "@/lib/actor/link"
import type { JobOrder } from "@/lib/domain/types"

export default async function LayarAksi({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const { id } = await params
  const actor = await actorFor("lapangan", first((await searchParams).actor))

  let order: JobOrder
  try {
    // Batas baca inspektor: order yang tidak ditugaskan kepadanya dijawab
    // tidak ditemukan oleh backend.
    order = await getOrder(actor.id, id)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) notFound()
    throw error
  }

  return <LapanganOrderDetail order={order} />
}
