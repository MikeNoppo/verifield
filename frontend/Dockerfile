# syntax=docker/dockerfile:1

# --- Tahap dependency --------------------------------------------------------
FROM oven/bun:1.4-alpine AS deps

WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

# --- Tahap build -------------------------------------------------------------
FROM oven/bun:1.4-alpine AS build

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# WARNING: NEXT_PUBLIC_* ikut terpanggang ke dalam bundel saat build, bukan
# dibaca saat container berjalan. Alamat backend karena itu harus disuntikkan di
# sini — mengubahnya nanti lewat environment variable tidak akan berpengaruh.
ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL

RUN bun run build

# --- Tahap runtime -----------------------------------------------------------
FROM oven/bun:1.4-alpine

WORKDIR /app
ENV NODE_ENV=production

RUN addgroup -g 10001 verifield && adduser -D -u 10001 -G verifield verifield

# Keluaran standalone hanya membawa berkas yang benar-benar dipakai saat jalan,
# jadi node_modules lengkap tidak ikut ke image runtime.
COPY --from=build --chown=verifield:verifield /app/.next/standalone ./
COPY --from=build --chown=verifield:verifield /app/.next/static ./.next/static
COPY --from=build --chown=verifield:verifield /app/public ./public

USER verifield
EXPOSE 3000
ENV HOSTNAME=0.0.0.0 PORT=3000

HEALTHCHECK --interval=15s --timeout=3s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/ >/dev/null || exit 1

CMD ["bun", "server.js"]
