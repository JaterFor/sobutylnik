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

## Deploying to the VPS

Server runs as `sobutylnik.service` (systemd) under `/opt/sobutylnik`, reverse-proxied by nginx on port 80 (no TLS yet — see design.md). Binaries are cross-compiled locally and copied over; there is no CI/CD pipeline yet.

```bash
# from backend/
GOOS=linux GOARCH=amd64 go build -o sobutylnik-server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o sobutylnik-migrate ./cmd/migrate

scp sobutylnik-server ubuntu@<server>:/opt/sobutylnik/sobutylnik-server
scp sobutylnik-migrate ubuntu@<server>:/opt/sobutylnik/sobutylnik-migrate
scp -r migrations/* ubuntu@<server>:/opt/sobutylnik/migrations/

ssh ubuntu@<server> 'sudo systemctl restart sobutylnik && sudo systemctl status sobutylnik'
```

`/opt/sobutylnik/.env` on the server holds real secrets (not in git). `TELEGRAM_BOT_TOKEN`, `R2_*`, and `EXPO_PUSH_ACCESS_TOKEN` are currently placeholders — Telegram login, photo upload, and push notifications won't work in production until those are filled in with real credentials.
