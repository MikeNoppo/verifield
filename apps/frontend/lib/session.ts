import "server-only"

import { cache } from "react"

import { listActors } from "@/lib/api/orders"
import { resolveActor } from "@/lib/actor/resolve"
import type { Actor, Persona } from "@/lib/domain/types"

/** Autentikasi berada di luar cakupan, sehingga identitas diambil dari daftar
    aktor contoh yang disediakan backend. Peran ditentukan segmen URL — bukan
    cookie maupun localStorage, karena keduanya berbagi nilai antar-tab dan itu
    membuat mustahil membuka satu tab sebagai inspektor dan satu lagi sebagai
    klien. Padahal justru itu yang memperagakan pembaruan langsung (F-01).
 *
 *  cache() menahan hasilnya selama satu render, jadi layout dan halaman di
 *  bawahnya tidak memanggil backend berulang kali. */
const actors = cache(listActors)

/** Seluruh aktor demo, untuk pemilih peran → aktor di kanan atas. */
export function demoActors(): Promise<Actor[]> {
  return actors()
}

export async function actorFor(persona: Persona, actorId?: string): Promise<Actor> {
  return resolveActor(await actors(), persona, actorId || null)
}
