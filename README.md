# braille-library

Backend and data layer of a digital library of Kazakh Braille texts.

The texts in the library are digitized by a Braille recognition model (computer vision and
deep learning, from the master's thesis "Kazakh Braille text recognition with computer vision
and deep learning"). Users read and listen to the texts, rate them, and correct recognition
errors. The corrections are stored as interactions and become training data for the model.

The project uses MongoDB as a NoSQL store with three collections: `users`, `items` and
`interactions`. Interactions are a separate collection on purpose: they grow without bound,
and embedding them in users or items would hit MongoDB's 16 MB document limit.
Recommendations are not part of this stage.

## Stack

Go, chi, MongoDB 8 (official Go driver v2), JWT (golang-jwt), bcrypt, Swagger (swaggo),
gofakeit (seed data), Docker Compose with mongo-express.

## Requirements

- Docker with Docker Compose
- Go 1.26+ (for the seed, the tests and running the API locally)
- Node.js 20+ (only to run the web UI outside Docker)
- `make` is optional: every target has a plain command next to it below

## Configuration

Copy the example file and set your own secrets:

```bash
cp .env.example .env                 # Linux / macOS / Git Bash
Copy-Item .env.example .env          # Windows PowerShell
```

| Variable | Description | Example |
|---|---|---|
| `PORT` | HTTP port of the API | `8080` |
| `MONGO_URI` | MongoDB connection string (Docker Compose overrides it for the api container) | `mongodb://localhost:27017` |
| `MONGO_DB` | Database name | `braille_library` |
| `JWT_SECRET` | Secret for signing tokens, **required** | a long random string |
| `JWT_TTL_HOURS` | Token lifetime in hours | `24` |
| `CORS_ORIGINS` | Web UI origins allowed to call the API, comma-separated | `http://localhost:3000,http://localhost:5173` |
| `MONGO_EXPRESS_USER` | mongo-express login (Docker Compose only) | `admin` |
| `MONGO_EXPRESS_PASSWORD` | mongo-express password (Docker Compose only) | a password |

`.env` is in `.gitignore`; only `.env.example` is committed.

## Running

### With Docker Compose

```bash
docker compose up -d --build         # make up
```

| Service | URL |
|---|---|
| Web UI | http://localhost:3000 |
| API | http://localhost:8080 |
| Swagger UI | http://localhost:8080/docs |
| mongo-express (basic auth from `.env`) | http://localhost:8081 |
| MongoDB (local machine only) | `mongodb://localhost:27017` |

Stop everything (the data stays in the `mongo_data` volume):

```bash
docker compose down                  # make down
```

### Locally (API on your machine, MongoDB in Docker)

```bash
docker compose up -d mongo
go run ./cmd/api                     # make run
```

The API reads `.env` from the current directory. Variables already set in the environment
take priority. Indexes are created at startup (this is idempotent).

## Seed data

```bash
go run ./cmd/seed --reset            # make seed ARGS=--reset
```

This fills the database with 50 users, 200 items (Kazakh and Russian titles) and about 3000
interactions. A fixed random seed is used, so every run produces the same data. The patterns
are realistic: learners mostly read beginner texts, users mostly read their interest
categories, and texts with lower recognition accuracy get more corrections.

Without `--reset` the seed refuses to run on a non-empty database. `--reset` deletes all
users, items and interactions first. It runs against `MONGO_URI` from `.env`, so MongoDB must
be running (`docker compose up -d mongo`).

### Demo accounts

| Email | Password | Role |
|---|---|---|
| `admin@demo.kz` | `Demo12345` | admin |
| `reader@demo.kz` | `Demo12345` | reader |

## API

JSON everywhere. Errors look like `{"error": "message"}`. Lists are paginated with `page`
(default 1) and `limit` (default 20, max 100). IDs are hex strings.

Protected endpoints need the header `Authorization: Bearer <token>`. You get a token from
`POST /auth/login`. In Swagger UI, click **Authorize** and enter `Bearer <token>`.

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | | API and MongoDB status |
| POST | `/auth/register` | | Register (role `reader`, `learner` or `teacher`, never `admin`) |
| POST | `/auth/login` | | Log in, returns a JWT |
| GET | `/users/me` | user | My profile |
| PUT | `/users/me` | user | Update name, language, reading level, interests, audio preference |
| GET | `/users/me/history?type=&page=&limit=` | user | My interactions, newest first, with item titles |
| GET | `/items?q=&category=&level=&language=&sort=new\|rating&page=&limit=` | | Search the catalog |
| GET | `/items/{id}` | | One item |
| POST | `/items` | admin | Create an item |
| PUT | `/items/{id}` | admin | Replace an item's fields |
| DELETE | `/items/{id}` | admin | Delete an item |
| POST | `/interactions` | user | Record an interaction |
| GET | `/docs/*` | | Swagger UI |

**Search.** `q` is a MongoDB `$text` search over title, description and tags, and results are
sorted by text score. The text index uses `default_language: "none"` because MongoDB has no
Kazakh stemming, so words must match exactly (case-insensitive).

**Interactions.** `type` is one of `view`, `read`, `listen`, `rate`, `favorite`, `correction`.

| Type | `value` | Notes |
|---|---|---|
| `read` | progress 0-100, required | |
| `rate` | 1-5, required | one per user and item: rating again updates it and recalculates the item's `avgRating` and `ratingsCount` |
| `favorite` | none | one per user and item; repeating it returns the existing one |
| `correction` | none | needs `correction: {position, original, suggested}` |
| `view`, `listen` | none | |

Example:

```bash
curl -X POST localhost:8080/interactions \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"itemId":"<item id>","type":"rate","value":5}'
```

`POST /interactions` returns 201 when it creates a document, and 200 when an existing rating
was updated or the favorite already existed.

## Web UI

A React + TypeScript single-page app in `web/` (Vite, react-router, plain CSS). Docker Compose
builds it and serves it with nginx:

1. `docker compose up -d --build`
2. If the database is empty, seed it and build the similarities:
   `go run ./cmd/seed --reset` and `go run ./cmd/recompute`
3. Open http://localhost:3000 and log in as `reader@demo.kz` / `Demo12345`

| Page | What it does |
|---|---|
| Log in, Register | account; registration also asks for reading preferences |
| Recommended (home) | "Recommended for you" with the reason for each text |
| Catalog | search with filters for category, level, language and sort |
| Text page | details; rate 1-5, add to favorites, save reading progress, listen (texts with audio), report a recognition error; "Similar texts" |
| History | all my actions, newest first, with a filter by type |
| Profile | name, interests, reading level, language, audio preference |

Opening a text page while logged in records a `view`.

**Accessibility.** Many users are blind or have low vision and use screen readers, so the UI
uses semantic HTML (landmarks, headings, lists, `fieldset`/`legend`) and every input has a
label. It works fully with the keyboard and has a "Skip to main content" link. A thick
yellow-and-black focus ring is visible on any background, and text contrast is at least 7:1.
After each navigation, focus moves to the page heading so screen readers announce the new page.
Status and error messages are announced through live regions. A text size switch (A / A+ /
A++) is in the header, and the choice is remembered. Titles and descriptions carry
`lang="kk"` or `lang="ru"` so screen readers use the right voice.

The browser calls the API directly, so the API must allow the UI's origin (`CORS_ORIGINS`).
The API address is set at build time with `VITE_API_URL` (a build arg in `docker-compose.yml`,
default `http://localhost:8080`).

Run it outside Docker with hot reload (the API must be running):

```bash
cd web
npm install
npm run dev                          # http://localhost:5173
```

## Data model and indexes

| Collection | Main fields | Indexes |
|---|---|---|
| `users` | email, passwordHash, name, role, preferredLanguage, readingLevel, interests, prefersAudio | `email` unique |
| `items` | title, description, category, level, language, brailleGrade, pages, author, tags, hasAudio, recognitionAccuracy, avgRating, ratingsCount | text (title, description, tags); `category + level`; `language`; `avgRating` desc |
| `interactions` | userId, itemId, type, value, correction, createdAt | `userId + createdAt` desc; `itemId + type`; unique partial `userId + itemId + type` for `rate` and `favorite` |

Users store only reading preferences, never disability or health information.

## Tests

```bash
go test ./...                        # make test
```

## Swagger docs

The generated files in `docs/` are committed. To regenerate them after changing the annotations:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.4
swag init -g cmd/api/main.go -o docs --parseInternal      # make swag
```

## Project structure

```
cmd/
  api/                  API entry point (config, Mongo, indexes, HTTP server)
  seed/                 seed data generator
internal/
  config/               environment variables and .env loading
  domain/               User, Item, Interaction, allowed values, errors
  auth/                 bcrypt passwords and JWT tokens
  repository/mongo/     MongoDB connection, indexes and queries
  service/              business logic and validation
  http/
    router.go           routes
    handlers/           HTTP handlers (with Swagger annotations)
    middleware/         authentication, role check and CORS
docs/                   generated Swagger spec
web/                    React web UI (Vite), its Dockerfile and nginx.conf
docker-compose.yml      mongo, mongo-express, api, web
Dockerfile              multi-stage build of the API
Makefile                shortcuts: up, down, run, seed, swag, test
.env.example            example configuration
```
