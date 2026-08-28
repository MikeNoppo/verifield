"use client"

import { MoonIcon, SunIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { setDarkTheme, useIsDark } from "@/hooks/use-theme"

export function ThemeToggle() {
  const dark = useIsDark()

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={() => setDarkTheme(!dark)}
      aria-label={dark ? "Ganti ke tema terang" : "Ganti ke tema gelap"}
    >
      {dark ? <MoonIcon /> : <SunIcon />}
    </Button>
  )
}
