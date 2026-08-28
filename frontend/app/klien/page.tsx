import Link from "next/link"
import { PlusIcon } from "lucide-react"

import { PageHeading } from "@/components/verifield/app-shell"
import { StatusBadge } from "@/components/verifield/status-badge"
import { StatusRail } from "@/components/verifield/status-rail"
import { buttonVariants } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { listClientOrders, type OrderFilter } from "@/lib/api/orders"
import { CURRENT_CLIENT } from "@/lib/mock/seed"
import { DEMO_NOW } from "@/lib/demo-time"
import { sejak, tanggalJam } from "@/lib/format"

const FILTER = [
  { key: "", label: "Semua" },
  { key: "berjalan", label: "Berjalan" },
  { key: "selesai", label: "Selesai" },
  { key: "bermasalah", label: "Bermasalah" },
] as const

function first(v: string | string[] | undefined): string {
  return Array.isArray(v) ? (v[0] ?? "") : (v ?? "")
}

export default async function KlienOrders({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const f = first((await searchParams).f)
  const bucket = (["berjalan", "selesai", "bermasalah"] as const).find((b) => b === f)
  const orders = listClientOrders({ bucket } as OrderFilter)

  return (
    <>
      <PageHeading
        title="Order Saya"
        meta={
          <>
            {CURRENT_CLIENT} · {orders.length} order
          </>
        }
        action={
          <Link
            href="/klien/permintaan-baru"
            className={buttonVariants({ size: "sm" })}
          >
            <PlusIcon data-icon="inline-start" />
            Permintaan Baru
          </Link>
        }
      />

      <div className="mb-4 flex flex-wrap gap-1.5">
        {FILTER.map((item) => (
          <Link
            key={item.key}
            href={item.key ? `/klien?f=${item.key}` : "/klien"}
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

      {orders.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Tidak ada order pada saringan ini</EmptyTitle>
            <EmptyDescription>
              Coba pilih saringan lain, atau buat permintaan inspeksi baru.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="overflow-hidden rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[104px]">Ref</TableHead>
                <TableHead>Objek &amp; Lokasi</TableHead>
                <TableHead className="hidden w-[132px] lg:table-cell">Jadwal</TableHead>
                <TableHead className="w-[228px]">Status</TableHead>
                <TableHead className="hidden w-[116px] text-right sm:table-cell">
                  Diperbarui
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((o) => (
                <TableRow key={o.id} className="h-12">
                  <TableCell className="align-middle">
                    <Link
                      href={`/klien/order/${o.id}`}
                      prefetch={false}
                      className="tabular font-mono text-xs font-medium hover:underline"
                    >
                      {o.ref}
                    </Link>
                  </TableCell>

                  <TableCell className="align-middle">
                    <Link
                      href={`/klien/order/${o.id}`}
                      prefetch={false}
                      className="flex flex-col gap-0.5"
                    >
                      <span className="truncate text-sm">{o.object}</span>
                      <span className="truncate text-[11px] text-muted-foreground">
                        {o.inspectionType} · {o.location}
                      </span>
                    </Link>
                  </TableCell>

                  <TableCell className="tabular hidden align-middle text-xs text-muted-foreground lg:table-cell">
                    {tanggalJam(o.scheduledAt)}
                  </TableCell>

                  {/* Rel menjawab "orangnya sudah berangkat belum" tanpa perlu
                      dibaca; badge di sebelahnya yang membawa nama statusnya. */}
                  <TableCell className="align-middle">
                    <div className="flex items-center gap-3">
                      <StatusRail order={o} className="w-20 shrink-0" />
                      <StatusBadge status={o.status} size="sm" />
                    </div>
                  </TableCell>

                  <TableCell className="tabular hidden text-right align-middle text-xs text-muted-foreground sm:table-cell">
                    {sejak(o.updatedAt, DEMO_NOW)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  )
}
