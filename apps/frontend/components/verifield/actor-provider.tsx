"use client"

import * as React from "react"

import type { Actor } from "@/lib/domain/types"

const ActorContext = React.createContext<Actor | null>(null)

/** Identitas ditentukan di server lalu diturunkan ke komponen klien lewat
    context, bukan diambil ulang di browser.
 *
 *  Alasannya: setiap mutasi harus membawa header identitas, dan mengalirkannya
 *  sebagai prop melewati setiap dialog akan mengotori seluruh pohon komponen
 *  hanya demi satu nilai yang tidak pernah berubah selama satu layar hidup. */
export function ActorProvider({
  actor,
  children,
}: {
  actor: Actor
  children: React.ReactNode
}) {
  return <ActorContext.Provider value={actor}>{children}</ActorContext.Provider>
}

export function useActor(): Actor {
  const actor = React.useContext(ActorContext)
  if (!actor) {
    throw new Error("useActor harus dipakai di dalam ActorProvider")
  }
  return actor
}
