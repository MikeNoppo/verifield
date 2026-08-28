# syntax=docker/dockerfile:1

# --- Tahap build -------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Dependency disalin lebih dulu agar lapisan ini ikut ter-cache selama go.mod
# dan go.sum belum berubah — tanpa itu setiap perubahan satu baris kode memicu
# unduh ulang seluruh dependency.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO dimatikan supaya binary-nya benar-benar statis dan bisa berjalan di image
# tanpa libc. Migrasi ikut ter-embed di binary migrate, sehingga image runtime
# tidak perlu membawa folder migrations/ maupun tool Atlas.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api     ./cmd/api  \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && go build -trimpath -ldflags="-s -w" -o /out/seeder  ./cmd/seeder

# --- Tahap runtime -----------------------------------------------------------
# Alpine, bukan distroless, semata karena HEALTHCHECK butuh sesuatu yang bisa
# menembak HTTP. Kalau health check dipindah ke orkestrator (Kubernetes memakai
# httpGet, bukan perintah di dalam container), distroless jadi pilihan lebih baik.
FROM alpine:3.21

RUN apk add --no-cache wget ca-certificates \
 && adduser -D -u 10001 verifield

COPY --from=build /out/api /out/migrate /out/seeder /usr/local/bin/

USER verifield
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["api"]
