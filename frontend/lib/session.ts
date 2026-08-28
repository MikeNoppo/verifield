import "server-only"

import { cache } from "react"

import { listActors } from "@/lib/api/orders"
import { PERSONA_ROLE, type Actor, type Persona } from "@/lib/domain/types"

/** Autentikasi berada di luar cakupan, sehingga identitas diambil dari daftar
    aktor contoh yang disediakan backend. Peran ditentukan segmen URL — bukan
    cookie maupun localStorage, karena keduanya berbagi nilai antar-tab dan itu
    membuat mustahil membuka satu tab sebagai inspektor dan satu lagi sebagai
    klien. Padahal justru itu yang memperagakan pembaruan langsung (F-01).
 *
 *  cache() menahan hasilnya selama satu render, jadi layout dan halaman di
 *  bawahnya tidak memanggil backend berulang kali. */
const actors = cache(listActors)

export async function actorFor(persona: Persona): Promise<Actor> {
  const role = PERSONA_ROLE[persona]
  const found = (await actors()).find((a) => a.role === role)

  if (!found) {
    throw new Error(
      `Tidak ada aktor contoh berperan "${role}". Jalankan seeder backend terlebih dahulu.`,
    )
  }
  return found
}
