# Autograph App

A digital autograph/guestbook app: owners organize entries into categories
(School, College, Company, Gym, ...), generate no-login share links for
guests to submit an autograph (name, note, photo, audio), review and approve
submissions, then view them like a flip-book.

```
autograph-app/
  backend/    Go API — auth, categories, share links, entries, S3 uploads
  frontend/   React (Vite) — owner dashboard + public guest submission page
```

## Quick start

**Backend**
```bash
cd backend
cp .env.example .env   # set JWT_SECRET at minimum
docker compose up -d    # local Postgres + MinIO (S3-compatible)
# psql "postgresql://postgres:postgres@localhost:5432/autograph?sslmode=disable" -f migrations/0001_init.up.sql
migrate -source file://migrations -database "postgres://postgres:postgres@localhost:5432/autograph?sslmode=disable" up
brew install minio-mc

mc alias set local http://localhost:9000 minioadmin minioadmin
mc anonymous set download local/autograph-uploads

export $(grep -v '^#' .env | xargs)
go run ./cmd/api
```

**Frontend**
```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Then visit http://localhost:5173, sign up as an owner, create a category,
generate a share link, and open that link in a private/incognito tab to try
the guest submission flow end-to-end.

Full details are in `backend/README.md` and `frontend/README.md`.

## What's here vs. what's next

Built: signup/login, category CRUD (with nesting), share-link generation,
public guest submission (with a preview/confirm step), owner moderation
queue, and a simple book view.

Not yet built (natural next steps): link expiry/usage-limit UI, multi-image
carousel in the book view, page-flip animation, email notifications when a
new submission comes in, and — once this is solid — a React Native app
reusing `frontend/src/api/client.js` almost as-is.
