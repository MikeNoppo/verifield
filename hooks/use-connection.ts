"use client"

import * as React from "react"

export type ConnectionState = "online" | "reconnecting" | "offline"

/** Status koneksi hidup di luar React, jadi ia disimpan sebagai external store
    dan bukan sebagai state yang disetel dari dalam effect. Transisi ke
    "menyambung ulang" ditangani di dalam listener, sehingga tidak ada setState
    sinkron yang memicu render berantai. */
let state: ConnectionState = "online"
let timer: ReturnType<typeof setTimeout> | null = null
const listeners = new Set<() => void>()

function set(next: ConnectionState) {
  if (state === next) return
  state = next
  for (const l of listeners) l()
}

function clearTimer() {
  if (timer === null) return
  clearTimeout(timer)
  timer = null
}

function handleOffline() {
  clearTimer()
  set("offline")
}

function handleOnline() {
  clearTimer()
  set("reconnecting")
  timer = setTimeout(() => {
    timer = null
    set("online")
  }, 1400)
}

function subscribe(onChange: () => void) {
  if (listeners.size === 0) {
    state = navigator.onLine ? "online" : "offline"
    window.addEventListener("online", handleOnline)
    window.addEventListener("offline", handleOffline)
  }
  listeners.add(onChange)

  return () => {
    listeners.delete(onChange)
    if (listeners.size > 0) return
    window.removeEventListener("online", handleOnline)
    window.removeEventListener("offline", handleOffline)
    clearTimer()
  }
}

const getSnapshot = () => state

// Server dan render pertama di klien sama-sama "online", sehingga markup
// keduanya identik. Nilai sebenarnya masuk setelah berlangganan.
const getServerSnapshot = (): ConnectionState => "online"

/** Sinyal koneksi asli dari browser. Layar yang diam punya dua arti yang tidak
    bisa dibedakan — pekerjaan memang belum berubah, atau sambungan putus — dan
    perbedaan itulah yang menentukan klien perlu menelepon atau tidak (F-01). */
export function useConnection(): ConnectionState {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}
