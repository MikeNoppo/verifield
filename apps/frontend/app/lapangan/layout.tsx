import { AppShell } from "@/components/verifield/app-shell"
import { ActorScope } from "@/components/verifield/actor-provider"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { LiveProvider } from "@/components/verifield/live-provider"
import { OutboxProvider } from "@/components/verifield/outbox-provider"
import { demoActors } from "@/lib/session"

export default async function LapanganLayout({ children }: LayoutProps<"/lapangan">) {
  const actors = await demoActors()

  return (
    <ActorScope persona="lapangan" actors={actors}>
      <LiveProvider>
        <OutboxProvider>
          <AppShell
            width="narrow"
            persona="lapangan"
              right={<ConnectionIndicator compact />}
          >
            {children}
          </AppShell>
        </OutboxProvider>
      </LiveProvider>
    </ActorScope>
  )
}
