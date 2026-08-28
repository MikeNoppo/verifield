import Link from "next/link"

import { cn } from "@/lib/utils"
import type { Persona } from "@/lib/domain/types"
import { RoleSwitcher } from "./role-switcher"
import { ShellIdentity } from "./shell-identity"
import { ThemeToggle } from "./theme-toggle"

const WIDTH = {
  wide: "max-w-[1600px]",
  normal: "max-w-6xl",
  narrow: "max-w-lg",
} as const

export function AppShell({
  width = "normal",
  persona,
  right,
  children,
}: {
  width?: keyof typeof WIDTH
  persona: Persona
  right?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-full flex-col">
      <header className="sticky top-0 z-30 border-b border-border bg-background/85 backdrop-blur">
        <div className={cn("mx-auto flex h-14 items-center gap-4 px-4 sm:px-6", WIDTH[width])}>
          <Link href="/" className="flex shrink-0 items-baseline gap-2">
            <span className="text-sm font-semibold tracking-tight">Verifield</span>
            <ShellIdentity persona={persona} />
          </Link>

          <div className="ml-auto flex items-center gap-3">
            {right}
            <RoleSwitcher persona={persona} />
            <ThemeToggle />
          </div>
        </div>
      </header>

      <main className={cn("mx-auto w-full flex-1 px-4 py-6 sm:px-6", WIDTH[width])}>
        {children}
      </main>
    </div>
  )
}

export function PageHeading({
  title,
  meta,
  action,
}: {
  title: string
  meta?: React.ReactNode
  action?: React.ReactNode
}) {
  return (
    <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
      <div className="flex flex-col gap-1">
        <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
        {meta ? <div className="text-xs text-muted-foreground">{meta}</div> : null}
      </div>
      {action}
    </div>
  )
}
