# syntax=docker/dockerfile:1
#
# WARNING: konteks build-nya adalah AKAR REPO, bukan direktori ini —
# lockfile dan paket kontrak berada di luar apps/frontend. Lihat
# docker-compose.yml; membangunnya dengan `docker build apps/frontend` akan gagal.

# --- Tahap build -------------------------------------------------------------
FROM oven/bun:1.4-alpine AS build

WORKDIR /repo

# Manifest disalin lebih dulu supaya lapisan install ter-cache selama daftar
# dependency belum berubah. Manifest backend ikut karena ia anggota workspace;
# tanpa itu bun menolak lockfile-nya sebagai tidak lengkap.
COPY package.json bun.lock ./
COPY apps/frontend/package.json apps/frontend/
COPY apps/backend/package.json apps/backend/
COPY packages/contract/package.json packages/contract/
RUN bun install --frozen-lockfile

COPY packages/contract packages/contract
COPY apps/frontend apps/frontend

# WARNING: NEXT_PUBLIC_* ikut terpanggang ke dalam bundel saat build, bukan
# dibaca saat container berjalan. Alamat backend karena itu harus disuntikkan di
# sini — mengubahnya nanti lewat environment variable tidak akan berpengaruh.
ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL

RUN cd apps/frontend && bun run build

# --- Tahap runtime -----------------------------------------------------------
FROM oven/bun:1.4-alpine

WORKDIR /app
ENV NODE_ENV=production

RUN addgroup -g 10001 verifield && adduser -D -u 10001 -G verifield verifield

# Keluaran standalone workspace mempertahankan struktur direktorinya, sehingga
# server.js berada di apps/frontend/ — bukan di akar seperti pada proyek tunggal.
COPY --from=build --chown=verifield:verifield /repo/apps/frontend/.next/standalone ./
COPY --from=build --chown=verifield:verifield /repo/apps/frontend/.next/static ./apps/frontend/.next/static
COPY --from=build --chown=verifield:verifield /repo/apps/frontend/public ./apps/frontend/public

USER verifield
EXPOSE 3000
ENV HOSTNAME=0.0.0.0 PORT=3000

HEALTHCHECK --interval=15s --timeout=3s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/ >/dev/null || exit 1

CMD ["bun", "apps/frontend/server.js"]
