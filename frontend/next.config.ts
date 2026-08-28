import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  typedRoutes: true,
  // Menghasilkan server mandiri berisi hanya berkas yang benar-benar dipakai
  // saat berjalan, sehingga image runtime tidak perlu membawa node_modules.
  output: "standalone",
};

export default nextConfig;
