import { AppShell } from "@/components/verifield/app-shell"
import { ActorScope } from "@/components/verifield/actor-provider"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { LiveProvider } from "@/components/verifield/live-provider"
import { demoActors } from "@/lib/session"

export default async function KlienLayout({ children }: LayoutProps<"/klien">) {
  const actors = await demoActors()

  return (
    <ActorScope persona="klien" actors={actors}>
      <LiveProvider>
        <AppShell
          persona="klien"
          right={<ConnectionIndicator className="hidden md:inline-flex" />}
        >
          {children}
        </AppShell>
      </LiveProvider>
    </ActorScope>
  )
}
