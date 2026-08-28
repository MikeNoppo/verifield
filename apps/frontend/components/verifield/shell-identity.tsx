"use client"

import { identityLabel } from "@/lib/actor/resolve"
import type { Persona } from "@/lib/domain/types"
import { useActor } from "./actor-provider"

/** Siapa yang sedang memakai layar ini — nama perusahaan klien, nama
    koordinator, atau kode inspektor. Tidak ada autentikasi di PoC.

    Dipisahkan dari AppShell supaya kerangka layar dan PageHeading tetap
    dirender di server: hanya potongan ini yang butuh aktor. */
export function ShellIdentity({ persona }: { persona: Persona }) {
  return (
    <span className="hidden text-[11px] text-muted-foreground sm:inline">
      {identityLabel(persona, useActor())}
    </span>
  )
}
