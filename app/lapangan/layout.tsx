import { AppShell } from "@/components/verifield/app-shell"
import { INSPECTORS, CURRENT_INSPECTOR_ID } from "@/lib/mock/seed"

export default function LapanganLayout({ children }: LayoutProps<"/lapangan">) {
  const me = INSPECTORS.find((i) => i.id === CURRENT_INSPECTOR_ID)!
  return (
    <AppShell width="narrow" identity={`${me.name} · ${me.code}`}>
      {children}
    </AppShell>
  )
}
