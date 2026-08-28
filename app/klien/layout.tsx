import { AppShell } from "@/components/verifield/app-shell"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { CURRENT_CLIENT } from "@/lib/mock/seed"
import { DEMO_NOW } from "@/lib/demo-time"
import { waktu } from "@/lib/format"

export default function KlienLayout({ children }: LayoutProps<"/klien">) {
  return (
    <AppShell
      identity={CURRENT_CLIENT}
      right={
        <ConnectionIndicator
          lastUpdate={waktu(DEMO_NOW.toISOString())}
          className="hidden md:inline-flex"
        />
      }
    >
      {children}
    </AppShell>
  )
}
