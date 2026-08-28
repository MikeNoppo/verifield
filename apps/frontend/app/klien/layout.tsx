import { AppShell } from "@/components/verifield/app-shell"
import { ActorProvider } from "@/components/verifield/actor-provider"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { LiveProvider } from "@/components/verifield/live-provider"
import { actorFor } from "@/lib/session"

export default async function KlienLayout({ children }: LayoutProps<"/klien">) {
  const actor = await actorFor("klien")

  return (
    <ActorProvider actor={actor}>
      <LiveProvider actorId={actor.id}>
        <AppShell
          identity={actor.companyName ?? actor.name}
          right={<ConnectionIndicator className="hidden md:inline-flex" />}
        >
          {children}
        </AppShell>
      </LiveProvider>
    </ActorProvider>
  )
}
