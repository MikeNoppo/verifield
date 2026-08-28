"use client"

import * as React from "react"
import { useSearchParams } from "next/navigation"

import { findActor } from "@/lib/actor/resolve"
import { PERSONA_ROLE, type Actor, type Persona } from "@/lib/domain/types"

type Scope = {
  actor: Actor
  /** Seluruh aktor demo, untuk pemilih aktor di kanan atas. Dititipkan di sini
      supaya kerangka layar tidak perlu meneruskannya sebagai prop, yang berarti
      daftar yang sama terkirim dua kali pada setiap muatan halaman. */
  actors: Actor[]
}

const ActorContext = React.createContext<Scope | null>(null)

/** Aktor diselesaikan di browser, bukan di layout server.
 *
 *  Layout App Router tidak menerima searchParams sama sekali — hanya params dan
 *  children — sehingga aktor yang dipilih lewat ?actor= tidak pernah sampai ke
 *  sana. Akibatnya kerangka layar tertinggal pada aktor baku peran sementara
 *  isi halaman sudah berpindah, dan pemilih aktor yang terkontrol memantul
 *  kembali ke nilai lama setiap kali dipilih.
 *
 *  Menyelesaikannya di klien juga menjaga LiveProvider tetap terpasang di
 *  layout: stream hanya tersambung ulang saat aktornya benar-benar berganti,
 *  bukan setiap kali berpindah halaman. */
export function ActorScope({
  persona,
  actors,
  children,
}: {
  persona: Persona
  actors: Actor[]
  children: React.ReactNode
}) {
  const actorId = useSearchParams().get("actor")
  const actor = React.useMemo(
    () => findActor(actors, persona, actorId),
    [actors, persona, actorId],
  )
  const scope = React.useMemo(() => (actor ? { actor, actors } : null), [actor, actors])

  if (!scope) return <MissingActor role={PERSONA_ROLE[persona]} />

  return <ActorContext.Provider value={scope}>{children}</ActorContext.Provider>
}

function MissingActor({ role }: { role: string }) {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-2 p-6 text-center">
      <p className="text-sm font-medium">Tidak ada aktor contoh berperan &ldquo;{role}&rdquo;</p>
      <p className="text-xs text-muted-foreground">Jalankan seeder backend terlebih dahulu.</p>
    </div>
  )
}

function useScope(): Scope {
  const scope = React.useContext(ActorContext)
  if (!scope) {
    throw new Error("useActor harus dipakai di dalam ActorScope")
  }
  return scope
}

export function useActor(): Actor {
  return useScope().actor
}

export function useActors(): Actor[] {
  return useScope().actors
}
