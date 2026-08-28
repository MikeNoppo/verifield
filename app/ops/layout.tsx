import { AppShell } from "@/components/verifield/app-shell"
import { ConnectionIndicator } from "@/components/verifield/connection-indicator"
import { DEMO_NOW } from "@/lib/demo-time"
import { waktu } from "@/lib/format"

export default function OpsLayout({ children }: LayoutProps<"/ops">) {
  return (
    <AppShell
      width="wide"
      identity="Rina Oktaviani · Koordinator Operasional"
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
