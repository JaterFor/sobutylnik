# sobutylnik backend

Go backend for the Sobutylnik MVP. See `/openspec/changes/mvp-core/` for specs and design.

## Local development

```bash
cp .env.example .env
# fill in DATABASE_URL, JWT secrets, etc.

go run ./cmd/migrate up
go run ./cmd/server
```

## Migrations

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate version
```
