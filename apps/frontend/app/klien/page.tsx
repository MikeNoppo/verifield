import Link from "next/link"
import { PlusIcon } from "lucide-react"

import { PageHeading } from "@/components/verifield/app-shell"
import { KlienOrderList } from "@/components/verifield/klien-order-list"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { listOrders } from "@/lib/api/orders"
import { actorFor } from "@/lib/session"
import { first, withActor } from "@/lib/actor/link"
import type { Bucket } from "@/lib/domain/summary"

const FILTER = [
  { key: "", label: "Semua" },
  { key: "berjalan", label: "Berjalan" },
  { key: "selesai", label: "Selesai" },
  { key: "bermasalah", label: "Bermasalah" },
] as const

export default async function KlienOrders({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const sp = await searchParams
  const f = first(sp.f)
  const bucket = (["berjalan", "selesai", "bermasalah"] as const).find((b) => b === f) as
    | Bucket
    | undefined

  const actor = await actorFor("klien", first(sp.actor))
  const href = (to: string) => withActor(to, actor.id)
  // Backend memaksakan saringan perusahaan sendiri untuk peran klien, apa pun
  // yang dikirim di query — kerahasiaan komersial antar klien tidak bergantung
  // pada perilaku halaman ini (asumsi A-03).
  const { orders } = await listOrders(actor.id, { limit: 100, sort_by: "status_changed_at" })

  return (
    <>
      <PageHeading
        title="Order Saya"
        meta={
          <>
            {actor.companyName} · {orders.length} order
          </>
        }
        action={
          <Link href={href("/klien/permintaan-baru")} className={buttonVariants({ size: "sm" })}>
            <PlusIcon data-icon="inline-start" />
            Permintaan Baru
          </Link>
        }
      />

      <div className="mb-4 flex flex-wrap gap-1.5">
        {FILTER.map((item) => (
          <Link
            key={item.key}
            href={href(item.key ? `/klien?f=${item.key}` : "/klien")}
            className={cn(
              "inline-flex h-7 items-center rounded-4xl border px-3 text-xs font-medium transition-colors",
              f === item.key
                ? "border-foreground/15 bg-foreground text-background"
                : "border-border text-muted-foreground hover:bg-muted/60 hover:text-foreground",
            )}
          >
            {item.label}
          </Link>
        ))}
      </div>

      <KlienOrderList orders={orders} bucket={bucket} />
    </>
  )
}
