import { AppShell } from "@/components/verifield/app-shell"
import { ActorProvider } from "@/components/verifield/actor-provider"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { LiveProvider } from "@/components/verifield/live-provider"
import { OutboxProvider } from "@/components/verifield/outbox-provider"
import { actorFor } from "@/lib/session"

export default async function LapanganLayout({ children }: LayoutProps<"/lapangan">) {
  const actor = await actorFor("lapangan")

  return (
    <ActorProvider actor={actor}>
      <LiveProvider actorId={actor.id}>
        <OutboxProvider>
          <AppShell width="narrow" identity={actor.name} right={<ConnectionIndicator compact />}>
            {children}
          </AppShell>
        </OutboxProvider>
      </LiveProvider>
    </ActorProvider>
  )
}
