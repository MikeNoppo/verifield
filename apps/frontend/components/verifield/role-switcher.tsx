"use client"

import Link from "next/link"
import { usePathname, useRouter, useSearchParams } from "next/navigation"
import type { Route } from "next"
import { BriefcaseIcon, HardHatIcon, LayoutDashboardIcon, UsersIcon } from "lucide-react"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useActor, useActors } from "@/components/verifield/actor-provider"
import { cn } from "@/lib/utils"
import { PERSONA_ROLE, type Actor, type Persona } from "@/lib/domain/types"

const ROLES = [
  { seg: "klien", label: "Klien", Icon: BriefcaseIcon },
  { seg: "ops", label: "Koordinator", Icon: LayoutDashboardIcon },
  { seg: "lapangan", label: "Inspektor", Icon: HardHatIcon },
] as const

function actorLabel(a: Actor): string {
  return a.companyName ? `${a.name} · ${a.companyName}` : a.name
}

/** Peran hidup di URL, bukan di cookie atau localStorage. Keduanya berbagi
    nilai antar-tab, sehingga mustahil membuka satu tab sebagai inspektor dan
    satu lagi sebagai klien — padahal justru itu yang membuktikan F-01. Aktor
    ikut hidup di query string (?actor=) karena alasan yang sama, dan karena
    query string ikut dikirim saat halaman dimuat ulang. */
export function RoleSwitcher({ persona, className }: { persona: Persona; className?: string }) {
  const actorId = useActor().id
  const actors = useActors()
  const pathname = usePathname()
  const search = useSearchParams()
  const router = useRouter()
  const [, aktif = ""] = pathname.split("/")

  const options = actors.filter((a) => a.role === PERSONA_ROLE[persona])

  // Baik berpindah peran maupun berganti aktor selalu mendarat di daftar peran
  // tujuan, tidak pernah di order yang sedang dibuka. Setiap peran dan setiap
  // aktor melihat himpunan order yang berbeda — inspektor hanya penugasannya
  // sendiri, klien hanya milik perusahaannya — sehingga membawa serta id order
  // mendaratkan sebagian perpindahan di layar "tidak ditemukan".
  function pilihAktor(v: string) {
    const params = new URLSearchParams(search.toString())
    params.set("actor", v)
    router.push(`/${aktif}?${params.toString()}` as Route)
  }

  return (
    <div
      className={cn(
        "inline-flex items-center gap-0.5 rounded-4xl border border-border bg-muted/40 p-0.5",
        className,
      )}
      role="group"
      aria-label="Lihat sebagai peran"
    >
      {ROLES.map(({ seg, label, Icon }) => {
        const current = aktif === seg
        return (
          <Link
            key={seg}
            href={`/${seg}` as Route}
            prefetch={false}
            aria-current={current ? "page" : undefined}
            className={cn(
              "inline-flex h-7 items-center gap-1.5 rounded-4xl px-2.5 text-xs font-medium transition-colors",
              current
                ? "bg-background text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Icon className="size-3.5" />
            <span className="hidden sm:inline">{label}</span>
          </Link>
        )
      })}

      {options.length > 1 ? (
        <span className="mx-0.5 flex items-center gap-1 border-l border-border/70 pl-1">
          <UsersIcon className="size-3.5 text-muted-foreground" />
          <Select value={actorId} onValueChange={(v) => pilihAktor(String(v))}>
            <SelectTrigger
              size="sm"
              className="h-7 max-w-44 border-0 bg-transparent px-1.5 shadow-none focus-visible:ring-2"
            >
              <SelectValue>
                {options.find((a) => a.id === actorId)?.name}
              </SelectValue>
            </SelectTrigger>
            <SelectContent align="end" className="w-auto min-w-(--anchor-width)">
              {options.map((a) => (
                <SelectItem key={a.id} value={a.id} label={actorLabel(a)}>
                  {actorLabel(a)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </span>
      ) : null}
    </div>
  )
}
