import Link from "next/link"
import { ArrowRightIcon, BriefcaseIcon, HardHatIcon, LayoutDashboardIcon } from "lucide-react"

import { ThemeToggle } from "@/components/verifield/theme-toggle"

const PERSONA = [
  {
    href: "/klien",
    label: "Klien",
    Icon: BriefcaseIcon,
    kondisi: "Komputer kantor, koneksi stabil, membuka sesekali",
    butuh:
      "Tahu inspektor sudah berangkat atau belum, tanpa menelepon siapa pun. Hanya melihat order milik perusahaannya.",
  },
  {
    href: "/ops",
    label: "Koordinator",
    Icon: LayoutDashboardIcon,
    kondisi: "Satu layar dipantau sepanjang hari",
    butuh:
      "Tahu secepat mungkin order mana yang bermasalah, tanpa memeriksa satu per satu. Menugaskan, mengoreksi, memutuskan pembatalan.",
  },
  {
    href: "/lapangan",
    label: "Inspektor",
    Icon: HardHatIcon,
    kondisi: "Ponsel, sambil berdiri, sarung tangan, sinyal tidak dapat diandalkan",
    butuh:
      "Memperbarui status dengan satu ketukan dan yakin laporannya tidak hilang meski sinyal sedang tidak ada.",
  },
] as const

export default function Home() {
  return (
    <div className="mx-auto flex min-h-full w-full max-w-4xl flex-col px-6 py-10">
      <header className="mb-10 flex items-start justify-between gap-4">
        <div className="flex flex-col gap-2">
          <h1 className="text-xl font-semibold tracking-tight">Verifield</h1>
          <p className="max-w-xl text-sm leading-relaxed text-muted-foreground">
            Pelacakan job order inspeksi lapangan untuk PT Sentra Inspeksi Nusantara. Status
            pekerjaan terlihat langsung oleh klien tanpa perantara manusia, dan inspektor punya
            cara memperbarui status yang tetap bekerja di area bersinyal buruk.
          </p>
        </div>
        <ThemeToggle />
      </header>

      <section className="mb-10">
        <h2 className="mb-3 text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Masuk sebagai
        </h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {PERSONA.map(({ href, label, Icon, kondisi, butuh }) => (
            <Link
              key={href}
              href={href}
              className="group flex flex-col gap-3 rounded-xl border border-border bg-card p-4 transition-colors hover:border-foreground/25 hover:bg-muted/40"
            >
              <span className="flex items-center gap-2">
                <Icon className="size-4 text-muted-foreground" />
                <span className="text-sm font-medium">{label}</span>
                <ArrowRightIcon className="ml-auto size-3.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
              </span>
              <span className="text-[11px] leading-relaxed text-muted-foreground">{kondisi}</span>
              <span className="text-xs leading-relaxed text-foreground/80">{butuh}</span>
            </Link>
          ))}
        </div>
      </section>

      <footer className="mt-auto border-t border-border pt-4 text-[11px] leading-relaxed text-muted-foreground">
        Autentikasi berada di luar cakupan, sehingga peran diwakili pemilih peran di kanan atas —
        bukan login. Data bersifat contoh dan dihitung relatif terhadap satu titik waktu tetap.
      </footer>
    </div>
  )
}
