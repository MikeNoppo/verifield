"use client"

import * as React from "react"

const KEY = "verifield-theme"

/** Sumber kebenaran tema adalah kelas .dark pada <html>, yang dipasang skrip di
    <head> sebelum React hidup. Karena itu ia dibaca sebagai external store —
    membacanya lewat effect akan memicu render kedua dan berbeda dari markup
    server. */
function subscribe(onChange: () => void) {
  const obs = new MutationObserver(onChange)
  obs.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] })
  return () => obs.disconnect()
}

const getSnapshot = () => document.documentElement.classList.contains("dark")
const getServerSnapshot = () => false

export function useIsDark(): boolean {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}

export function setDarkTheme(dark: boolean) {
  document.documentElement.classList.toggle("dark", dark)
  try {
    localStorage.setItem(KEY, dark ? "dark" : "light")
  } catch {
    // Mode privat bisa menolak penulisan; tema tetap berganti untuk sesi ini.
  }
}
