.PHONY: help run dev build tidy fmt vet test swagger clean \
	migrate-diff migrate-up migrate-down migrate-status migrate-version migrate-hash seed

MAIN          := ./cmd/api
BINARY        := bin/verifield-be
MIGRATE_BIN   := bin/migrate
SEEDER_BIN    := bin/seeder

# Muat .env supaya ATLAS_DEV_URL terbaca oleh perintah `atlas` tanpa perlu export manual.
ifneq (,$(wildcard .env))
include .env
export
endif

help: ## Tampilkan daftar perintah
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ---------------------------------------------------------------- aplikasi

run: ## Jalankan server
	go run $(MAIN)

dev: ## Jalankan server dengan hot reload (butuh: go install github.com/air-verse/air@latest)
	air

build: ## Build seluruh binary ke bin/
	go build -o $(BINARY) $(MAIN)
	go build -o $(MIGRATE_BIN) ./cmd/migrate
	go build -o $(SEEDER_BIN) ./cmd/seeder

## ---------------------------------------------------------------- migrasi

migrate-diff: ## Generate migrasi dari perubahan di internal/schema (~ prisma migrate dev). Pakai: make migrate-diff name=add_products
ifndef name
	$(error nama migrasi wajib diisi. Contoh: make migrate-diff name=add_products)
endif
	atlas migrate diff $(name) --env gorm

migrate-up: ## Terapkan seluruh migrasi yang tertunda (~ prisma migrate deploy)
	go run ./cmd/migrate up

migrate-down: ## Batalkan satu migrasi terakhir
	go run ./cmd/migrate down

migrate-status: ## Tampilkan migrasi yang sudah & belum diterapkan
	go run ./cmd/migrate status

migrate-version: ## Tampilkan versi migrasi database saat ini
	go run ./cmd/migrate version

migrate-hash: ## Perbarui atlas.sum setelah berkas migrasi diedit manual
	atlas migrate hash --env gorm

schema-print: ## Cetak DDL yang dihasilkan dari internal/schema (tanpa database)
	go run ./cmd/atlas-loader

seed: ## Buat admin pertama dari SEED_ADMIN_* di .env (~ prisma db seed)
	go run ./cmd/seeder

## ---------------------------------------------------------------- kualitas

tidy: ## Rapikan dan unduh dependency
	go mod tidy

fmt: ## Format seluruh kode
	go fmt ./...

vet: ## Analisis statis bawaan Go
	go vet ./...

test: ## Jalankan seluruh test
	go test ./... -race -cover

swagger: ## Generate dokumentasi Swagger (butuh: go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

clean: ## Hapus hasil build
	rm -rf bin tmp
