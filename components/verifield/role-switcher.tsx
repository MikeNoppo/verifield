"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import type { Route } from "next"
import { BriefcaseIcon, HardHatIcon, LayoutDashboardIcon } from "lucide-react"

import { cn } from "@/lib/utils"

const ROLES = [
  { seg: "klien", label: "Klien", Icon: BriefcaseIcon },
  { seg: "ops", label: "Koordinator", Icon: LayoutDashboardIcon },
  { seg: "lapangan", label: "Inspektor", Icon: HardHatIcon },
] as const

/** Peran hidup di URL, bukan di cookie atau localStorage. Keduanya berbagi
    nilai antar-tab, sehingga mustahil membuka satu tab sebagai inspektor dan
    satu lagi sebagai klien — padahal justru itu yang membuktikan F-01. */
export function RoleSwitcher({ className }: { className?: string }) {
  const pathname = usePathname()
  const [, aktif = "", ...sisa] = pathname.split("/")

  // Konteks order dibawa lintas peran: dari layar inspektor, satu klik
  // memperlihatkan order yang sama sebagaimana dilihat klien.
  const tail = sisa[0] === "order" && sisa[1] ? `/order/${sisa[1]}` : ""

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
            href={`/${seg}${tail}` as Route}
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
    </div>
  )
}
