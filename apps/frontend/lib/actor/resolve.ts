import { PERSONA_ROLE, type Actor, type Persona } from "@/lib/domain/types"

/** Dipakai server dan klien dari satu sumber: halaman menyelesaikan aktor dari
    searchParams-nya sendiri, sedangkan kerangka layar menyelesaikannya dari URL
    di browser. Aturan yang berbeda di antara keduanya berarti header dan isi
    layar bisa memperlihatkan aktor yang tidak sama. */
export function findActor(
  actors: Actor[],
  persona: Persona,
  actorId: string | null,
): Actor | undefined {
  const role = PERSONA_ROLE[persona]

  if (actorId) {
    const chosen = actors.find((a) => a.id === actorId && a.role === role)
    if (chosen) return chosen
  }
  return actors.find((a) => a.role === role)
}

/** Bentuk yang melempar, untuk pemanggil di server: di sana Next masih bisa
    merender halaman error. Komponen klien memakai findActor dan menangani
    ketiadaannya sendiri, karena lemparan di tengah render melepas seluruh
    pohon di bawahnya tanpa ada yang menampung. */
export function resolveActor(actors: Actor[], persona: Persona, actorId: string | null): Actor {
  const found = findActor(actors, persona, actorId)

  if (!found) {
    throw new Error(
      `Tidak ada aktor contoh berperan "${PERSONA_ROLE[persona]}". Jalankan seeder backend terlebih dahulu.`,
    )
  }
  return found
}

export function identityLabel(persona: Persona, actor: Actor): string {
  switch (persona) {
    case "klien":
      return actor.companyName ?? actor.name
    case "ops":
      return `${actor.name} · Koordinator Operasional`
    case "lapangan":
      return actor.name
  }
}
