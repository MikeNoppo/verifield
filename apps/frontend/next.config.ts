import path from "node:path"

import type { NextConfig } from "next"

const nextConfig: NextConfig = {
  typedRoutes: true,
  // Menghasilkan server mandiri berisi hanya berkas yang benar-benar dipakai
  // saat berjalan, sehingga image runtime tidak perlu membawa node_modules.
  output: "standalone",
  // Di dalam workspace, dependency tinggal di node_modules akar repo. Tanpa
  // akar penelusuran yang eksplisit, Next hanya menyusuri isi apps/frontend dan
  // keluaran standalone-nya kehilangan paket yang dibutuhkan saat berjalan.
  outputFileTracingRoot: path.join(import.meta.dirname, "../.."),
}

export default nextConfig
