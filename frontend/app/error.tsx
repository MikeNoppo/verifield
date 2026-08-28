"use client"

import { Button } from "@/components/ui/button"

export default function Error({ retry }: { error: Error & { digest?: string }; retry: () => void }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <h1 className="text-base font-semibold">Halaman gagal dimuat</h1>
      <p className="max-w-sm text-sm leading-relaxed text-muted-foreground">
        Terjadi kesalahan saat menyiapkan tampilan. Data Anda tidak terpengaruh.
      </p>
      <Button onClick={() => retry()} variant="secondary" className="mt-2">
        Coba lagi
      </Button>
    </div>
  )
}
