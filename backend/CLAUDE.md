# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Environment gotchas

- **This package lives at `backend/` inside a monorepo.** The frontend sits beside it at `frontend/`, and `docs/` at the repo root is the shared business specification. Run every command in this file from `backend/`.
- **`make` is not installed.** The Makefile documents the intended workflow, but you must run the underlying `go` commands directly.
- **PostgreSQL runs in Docker, not natively; there is no `psql` binary.** Start it with `docker compose up -d postgres` from the repo root before anything that touches a database — the server, `migrate up`, the seeder, `atlas migrate diff`. To run SQL, go through the container (`docker compose exec postgres psql ...`).
- **Atlas is not installed.** Needed only to *generate* migrations. Install with `go install ariga.io/atlas/cmd/atlas@latest`, or fall back to `go run ./cmd/atlas-loader`, which prints the full DDL offline so a goose file can be written by hand.
- `swag` is installed at `~/go/bin/swag`.

## Commands

```bash
go build ./... && go vet ./... && gofmt -l .   # gofmt -l prints offenders; empty = clean
go test ./...                                  # runs without a database
go test ./internal/app/ -run TestValidasiDTOGagal -v # single test

go run ./cmd/atlas-loader                       # prints the DDL from schema.go — works offline, no DB
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

Regenerate Swagger whenever controller annotations change; `docs/` is committed.

## Language convention

Code comments, commit messages, and all user-facing strings (API messages, validation errors, CLI output) are written in **Indonesian**. Match this — do not introduce English strings into the response envelope or error messages.

## Architecture

Go + Gin + GORM/PostgreSQL, deliberately shaped to feel like **NestJS** (modules) with a **Prisma**-style single-file schema. The user comes from NestJS/Next.js and Prisma; explanations and naming should lean on those analogies.

### Module pattern

Each feature is a package under `internal/modules/` exposing `Module`, `NewModule(deps)`, and `RegisterRoutes(rg, guards...)`. Wiring is manual (no DI container): `repository → service → controller`. Cross-module dependencies are injected, never constructed — `auth.NewModule(jwtManager, userModule.Service)`.

Adding a module touches exactly four places:
1. table in `internal/schema/schema.go` + registered in `All()`
2. the new `internal/modules/xxx/` package
3. assembly in `app.New` ([internal/app/app.go](internal/app/app.go))
4. one `RegisterRoutes` line in [internal/app/router.go](internal/app/router.go)

Use `internal/modules/user/` as the reference implementation.

### Error handling contract — the most important rule

**Controllers never build error responses.** They call `c.Error(err); return`. [middleware/error_handler.go](internal/common/middleware/error_handler.go) runs after `c.Next()` and maps: `*apperror.AppError` → its own status, `validator.ValidationErrors` → 422 with per-field detail, `gorm.ErrRecordNotFound` → 404, `gorm.ErrDuplicatedKey` → 409, anything else → 500 with the real cause logged but never sent to the client.

Services return `apperror.NotFound(...)`, `apperror.Conflict(...)` etc. Successful responses always go through `response.OK/Created/Paginated/NoContent` so the JSON envelope stays uniform.

### Schema and migrations

`internal/schema/schema.go` is the single source of truth for every table — the Prisma analogue. It is a **leaf package**: it must never import other application packages, which is what keeps cross-module relations from creating import cycles. `schema.All()` is read by `cmd/atlas-loader`, which Atlas invokes to compute diffs.

GORM `AutoMigrate` has been removed on purpose. The pipeline is **Atlas generates, goose applies**:
- `atlas migrate diff <name> --env gorm` writes goose-annotated SQL into `migrations/`
- `cmd/migrate` applies them; `migrations/embed.go` embeds the SQL into that binary, so production needs neither Atlas nor the `migrations/` folder

Atlas needs an empty scratch database (`ATLAS_DEV_URL`) and never touches production.

### Security-relevant invariants

- `sort_by` is filtered against each repository's `SortableColumns` whitelist before reaching `ORDER BY`. Never interpolate a raw sort column.
- `user.Service` has two method families: DTO-returning ones for HTTP, and `*Entity` ones (`CreateEntity`, `FindEntityByEmail`, `FindEntityByID`) that return `*schema.User` **including the password hash**. Entity values must never be serialized into a response — map through `dto.ToUserResponse` first.
- Login failures return one generic message for both unknown email and wrong password, so email enumeration stays impossible.
- `validation.Init()` must run at bootstrap; it makes validation errors report JSON field names (`email`) instead of Go field names (`Email`).

## Tests

Both suites run without a database:
- [internal/app/smoke_test.go](internal/app/smoke_test.go) — HTTP end-to-end against a stub `user.Service`. Covers the guard chain, validation envelope, and that password never leaks. Copy this pattern for new module tests.
- [migrations/migrations_test.go](migrations/migrations_test.go) — asserts every migration file has a non-empty `-- +goose Down` block, guarding against Atlas emitting Up-only files that would silently break rollback.
