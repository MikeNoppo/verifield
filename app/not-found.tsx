import Link from "next/link"

import { Button } from "@/components/ui/button"

export default function NotFound() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <h1 className="text-base font-semibold">Order tidak ditemukan</h1>
      <p className="max-w-sm text-sm leading-relaxed text-muted-foreground">
        Nomor referensi ini tidak ada, atau bukan milik perusahaan yang sedang Anda gunakan.
        Klien hanya dapat melihat order milik perusahaannya sendiri.
      </p>
      <Button render={<Link href="/" />} variant="secondary" className="mt-2">
        Kembali ke awal
      </Button>
    </div>
  )
}
