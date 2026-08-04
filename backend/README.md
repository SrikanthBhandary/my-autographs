# Autograph App — Backend (Go)

Plain Go (net/http + Go 1.22's built-in method/path routing — no framework),
Postgres via `database/sql` + `lib/pq`, JWT auth for owners, S3-compatible
storage for images/audio, and a token-gated public API for guest submissions.

## 1. Prerequisites

- Go 1.22+
- Docker (for local Postgres + MinIO), or your own Postgres + S3 bucket

## 2. Local setup

```bash
cp .env.example .env
# edit .env — at minimum set a real JWT_SECRET

docker compose up -d          # starts Postgres :5432, MinIO :9000/:9001, Mailpit :1025/:8025, RabbitMQ :5672/:15672
```

If using the bundled MinIO for local dev, set these in `.env`:
```
S3_ENDPOINT=http://localhost:9000
S3_USE_PATH_STYLE=true
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_PUBLIC_BASE_URL=http://localhost:9000/autograph-uploads
```
Then create the bucket once via the MinIO console at http://localhost:9001
(login minioadmin/minioadmin), or with the `mc` CLI.

### Email notifications

Owners get an email when a guest leaves a new autograph, so they know to go
approve it. It's SMTP-based, so it works with any provider — see the
commented options in `.env.example` (Gmail, SendGrid, Mailgun, Postmark, SES,
Resend).

For local dev, the bundled **Mailpit** container needs no credentials at
all — `.env.example`'s defaults already point at it. Every notification
email lands at **http://localhost:8025** instead of a real inbox.

If `SMTP_HOST` is left blank, notifications are silently disabled (logged at
startup, then skipped per-send) — nothing else breaks.

## 3. Run the database migration

```bash
psql "postgresql://postgres:postgres@localhost:5432/autograph?sslmode=disable" \
  -f migrations/0001_init.up.sql
```

(Or install [golang-migrate](https://github.com/golang-migrate/migrate) if
you want proper up/down migration tracking as the schema evolves.)

## 4. Run the server

```bash
go run ./cmd/api
```

The app loads `.env` automatically on startup (via `github.com/joho/godotenv`),
so as long as `.env` sits next to `go.mod` in `backend/`, no manual exporting
is needed. In production, just set real environment variables instead — the
`.env` file won't exist there, which is fine, `godotenv.Load()` silently no-ops.

Server starts on `:8080` (or `$PORT`).

### PDF export (worker process)

Exporting a book to PDF is async: the API queues a job in RabbitMQ and
returns immediately; a separate **worker** process actually builds the PDF
and notifies the browser over WebSocket when it's ready. Run it alongside
the API:

```bash
go run ./cmd/worker
```

You can run more than one worker process for throughput — RabbitMQ spreads
jobs across whichever workers are running, no config needed. If RabbitMQ
isn't reachable at API startup, export is disabled gracefully (logged once,
then the "Export as PDF" button returns a 503) — nothing else breaks.

Watch jobs move through the queue at the RabbitMQ management UI:
http://localhost:15672 (login `guest`/`guest` for the bundled dev instance).

## 5. API overview

| Route | Auth | Purpose |
|---|---|---|
| `POST /api/auth/signup` | none | create owner account |
| `POST /api/auth/login` | none | get JWT |
| `GET/POST /api/categories` | JWT | list/create categories (School, College, Company, Gym...) |
| `PUT/DELETE /api/categories/{id}` | JWT | edit/remove a category |
| `POST /api/sharelinks` | JWT | generate a shareable guest link for a category |
| `PATCH /api/sharelinks/{id}/deactivate` | JWT | kill a link early |
| `POST /api/submit/{token}` | share-link token only | **guest** submits an autograph (multipart form: `guest_name`, `note`, `images[]`, `audio`) |
| `GET /api/entries?status=pending` | JWT | moderation queue |
| `PATCH /api/entries/{id}/approve` | JWT | approve → visible in book |
| `PATCH /api/entries/{id}/reject` | JWT | reject |
| `GET /api/entries?status=approved` | JWT | data for the book view |
| `POST /api/export/pdf` | JWT | queue a PDF export job (`category_id` optional — omit for the whole book) |
| `GET /api/export/pdf/{id}` | JWT | poll a job's status (fallback if the WebSocket notification is missed) |
| `GET /api/ws?token=<jwt>` | JWT (query param) | WebSocket — pushes `{job_id, status, file_url}` when an export finishes |

## 6. Deploying

Any host that runs a Go binary works (Fly.io, Railway, Render, a VPS, ECS).
Build static binaries with:

```bash
CGO_ENABLED=0 go build -o api ./cmd/api
CGO_ENABLED=0 go build -o worker ./cmd/worker
```

Point `DB_*` at a managed Postgres (RDS, Supabase, Neon, etc.), `S3_*` at
real S3 or Cloudflare R2, and `RABBITMQ_URL` at a hosted broker (e.g.
CloudAMQP's free tier works fine to start) — no code changes needed, just
env vars. `api` and `worker` are separate deployable processes; run at
least one of each, and scale `worker` up independently if export volume
grows.
