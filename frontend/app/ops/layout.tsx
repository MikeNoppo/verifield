import { AppShell } from "@/components/verifield/app-shell"
import { ActorProvider } from "@/components/verifield/actor-provider"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { LiveProvider } from "@/components/verifield/live-provider"
import { actorFor } from "@/lib/session"

export default async function OpsLayout({ children }: LayoutProps<"/ops">) {
  const actor = await actorFor("ops")

  return (
    <ActorProvider actor={actor}>
      <LiveProvider actorId={actor.id}>
        <AppShell
          width="wide"
          identity={`${actor.name} · Koordinator Operasional`}
          right={<ConnectionIndicator className="hidden md:inline-flex" />}
        >
          {children}
        </AppShell>
      </LiveProvider>
    </ActorProvider>
  )
}
