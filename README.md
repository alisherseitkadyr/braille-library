# braille-library

A digital library of Kazakh Braille texts with a recommendation system on MongoDB.

The texts in the library are digitized by a Braille recognition model (computer vision and
deep learning, from the master's thesis "Kazakh Braille text recognition with computer vision
and deep learning"). Users read and listen to the texts, rate them, and correct recognition
errors. The corrections are stored as interactions and become training data for the model.
From the same interactions the system recommends texts with item-based collaborative
filtering.

MongoDB is the NoSQL store: `users`, `items`, `interactions` and `item_similarities`.
Interactions are a separate collection on purpose: they grow without bound, and embedding
them in users or items would hit MongoDB's 16 MB document limit.

## Quick start

Needs only Docker with Docker Compose.

```bash
cp .env.example .env                 # Windows PowerShell: Copy-Item .env.example .env
docker compose up -d --build         # make up
```

Open http://localhost:3000 and log in as `reader@demo.kz` / `Demo12345`.

On the first start the `init` job loads the demo data into the empty database and builds the
item similarities, then the API starts. The first start takes a few minutes, mostly to build
the images (40 s when Docker already has the base images cached). Later starts keep your
data and only rebuild the similarities.

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
docker compose down -v               # also delete the data
```

## Stack

Go, chi, MongoDB 8 (official Go driver v2), JWT (golang-jwt), bcrypt, Swagger (swaggo),
gofakeit (seed data); React, TypeScript, Vite, react-router (web UI, served by nginx);
k6 (load tests); Docker Compose with mongo-express.

## Requirements

- Docker with Docker Compose: enough to run everything
- Go 1.26+: for the tests, the seed with custom sizes and running the API outside Docker
- k6 and bash (Git Bash on Windows): for the performance tests
- Node.js 20+: only to run the web UI outside Docker
- `make` is optional: every target has a plain command next to it below

## Configuration

`.env` is read by Docker Compose and by the Go commands. It is in `.gitignore`; only
`.env.example` is committed. Change `JWT_SECRET` and `MONGO_EXPRESS_PASSWORD` for anything
other than a local demo.

| Variable | Description | Default / example |
|---|---|---|
| `PORT` | HTTP port of the API | `8080` |
| `MONGO_URI` | MongoDB connection string (Docker Compose overrides it inside the containers) | `mongodb://localhost:27017` |
| `MONGO_DB` | Database name | `braille_library` |
| `JWT_SECRET` | Secret for signing tokens, **required** | a long random string |
| `JWT_TTL_HOURS` | Token lifetime in hours | `24` |
| `CORS_ORIGINS` | Web UI origins allowed to call the API, comma-separated | `http://localhost:3000,http://localhost:5173` |
| `RECS_MODE` | `precomputed` (reads `item_similarities`) or `live` (computes similarities on every request, the baseline) | `precomputed` |
| `CACHE_TTL` | How long recommendations are cached per user; `0` turns the cache off | `60s` |
| `MONGO_EXPRESS_USER` | mongo-express login (Docker Compose only) | `admin` |
| `MONGO_EXPRESS_PASSWORD` | mongo-express password (Docker Compose only) | a password |

Variables already set in the environment take priority over `.env`.

### Running the API outside Docker

```bash
docker compose up -d mongo
go run ./cmd/api                     # make run
```

Indexes are created at startup (this is idempotent).

## Seed data

`docker compose up` loads the demo data automatically when the database is empty. To reload
it, or to generate big data sets, run the seed on your machine (MongoDB must be running):

```bash
go run ./cmd/seed --reset            # make seed ARGS=--reset
go run ./cmd/recompute               # make recompute: rebuild the similarities afterwards
```

The demo data is 50 users, 200 items (Kazakh and Russian titles) and about 3000
interactions. A fixed random seed is used, so every run produces the same data. The patterns
are realistic: learners mostly read beginner texts, users mostly read their interest
categories, and texts with lower recognition accuracy get more corrections.

| Flag | Meaning |
|---|---|
| `--reset` | delete all users, items, interactions and similarities first; without it the seed refuses to run on a non-empty database |
| `--if-empty` | do nothing if the database already has users (used by the `init` job) |
| `--users N --items N` | sizes (default 50 and 200) |
| `--interactions N` | exactly N interactions for performance tests; the generated users get the emails `user1@perf.kz`, `user2@perf.kz`, ... (password `Demo12345`) |

Example: `go run ./cmd/seed --reset --users 10000 --items 5000 --interactions 1000000`
(about 30 s, inserted in bulk batches of 10,000).

### Demo accounts

| Email | Password | Role |
|---|---|---|
| `admin@demo.kz` | `Demo12345` | admin |
| `reader@demo.kz` | `Demo12345` | reader |

## API

JSON everywhere. Errors look like `{"error": "message"}`. Lists are paginated with `page`
(default 1) and `limit` (default 20, max 100). IDs are hex strings. Full reference with
examples: Swagger UI at http://localhost:8080/docs.

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
| GET | `/users/me/recommendations?limit=` | user | Recommended items with score and reason (limit default 10, max 50) |
| GET | `/items?q=&category=&level=&language=&sort=new\|rating&page=&limit=` | | Search the catalog |
| GET | `/items/{id}` | | One item |
| GET | `/items/{id}/similar?limit=` | | Most similar items (limit default 10, max 20) |
| POST | `/items` | admin | Create an item |
| PUT | `/items/{id}` | admin | Replace an item's fields |
| DELETE | `/items/{id}` | admin | Delete an item |
| POST | `/interactions` | user | Record an interaction |
| POST | `/admin/recommendations/rebuild` | admin | Rebuild `item_similarities`, returns `{items, durationMs}` |
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

```bash
curl -X POST localhost:8080/interactions \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"itemId":"<item id>","type":"rate","value":5}'
```

`POST /interactions` returns 201 when it creates a document, and 200 when an existing rating
was updated or the favorite already existed.

## Recommendations

Item-based collaborative filtering: "people who liked the texts you liked also liked these".

### Algorithm

**1. User-item score s(u,i).** Each interaction has a weight:

| Interaction | Weight |
|---|---|
| view | 1 |
| listen | 2 |
| correction | 1 |
| read | 1.5, or 3 if progress >= 80 |
| favorite | 4 |
| rate 1 / 2 / 3 / 4 / 5 | -2 / -1 / 1 / 3 / 5 |

If the user rated the item, s(u,i) is the rating weight: explicit feedback overrides
everything else, so a text read to the end but rated 1 is disliked. Otherwise s(u,i) is the
maximum weight of the user's other interactions with the item. Only s(u,i) > 0 counts as
"liked". MongoDB computes the scores with one aggregation (`$group` by user and item, with the
weights as a `$switch` expression built from the same constants as the Go code).

**2. Item similarity.** Cosine similarity over the users who liked both items, with shrinkage
so that pairs with few common users count less:

```
sim(i,j) = sum_u s(u,i)*s(u,j) / (||i|| * ||j||) * n_ij / (n_ij + 10)
```

`n_ij` is the number of common users, `||i||` is the square root of the sum of s(u,i)² over
the users who liked i. Each item keeps its top 20 neighbors with sim > 0.

**3. Recommendations for user u.** Take the user's last 50 liked items. Every neighbor j of
those items gets `score(j) = sum_i sim(i,j) * s(u,i)`. Items the user already interacted with
(in any way) are excluded. The top N are returned with the reason "because you read
&lt;title&gt;", where the title is the history item that contributed most to the score.

**4. Cold start.** If there are fewer than N results (new users, or few neighbors), the list
is filled with items that match the profile (category in the user's interests, same reading
level and language), most popular first, reason "matches your interests"; then with the
globally most popular items, reason "popular". Popularity is the number of ratings, then the
average rating.

### Rebuild and modes

- **precomputed** (default): the neighbors of all items are stored in `item_similarities`
  (`{itemId, neighbors: [{itemId, score}], updatedAt}`, unique index on `itemId`). The rebuild
  job loads the liked scores with the aggregation, computes all similarities in Go and writes
  them with bulk upserts; documents of items that lost all neighbors are deleted. Run it with
  `go run ./cmd/recompute` (`make recompute`), `POST /admin/recommendations/rebuild`, or by
  restarting Docker Compose (the `init` job). New interactions affect the user's own
  recommendations at once (their history is read on every request), but the similarities
  only change at the next rebuild.
- **live** (`RECS_MODE=live`): the neighbors of the user's history items are computed on every
  request with two aggregations (co-occurrence of each history item with all other items, and
  the norms of all items involved). No `item_similarities`, no cache. It gives the same
  results as precomputed (checked on the demo data) and is the baseline of the performance
  tests.
- **cache** (`CACHE_TTL`, default 60 s): the recommendation list is cached in memory per user.
  The user's entry is dropped when they record an interaction, and the whole cache is
  cleared after a rebuild.

## Web UI

A React + TypeScript single-page app in `web/` (Vite, react-router, plain CSS), built and
served by nginx on http://localhost:3000.

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
| `items` | title, description, category, level, language, brailleGrade, pages, author, tags, hasAudio, recognitionAccuracy, avgRating, ratingsCount | text (title, description, tags); `language`; `avgRating` desc; `popularity`; `profile_popularity` |
| `interactions` | userId, itemId, type, value, correction, createdAt | `user_createdAt_id`; `item_type`; `user_item_scores`; unique partial `userId + itemId + type` for `rate` and `favorite` |
| `item_similarities` | itemId, neighbors [{itemId, score}], updatedAt | `itemId` unique |

The indexes added in the optimization step, and why:

| Index | Keys | Used by |
|---|---|---|
| `popularity` | `ratingsCount -1, avgRating -1, _id 1` | popular items; the index order is the sort order, so no in-memory sort and the scan stops after `limit` |
| `profile_popularity` | `category, level, language, ratingsCount -1, avgRating -1, _id 1` | items matching the profile. ESR rule: equality fields first, then the sort; with several interests MongoDB merges the sorted ranges (SORT_MERGE) |
| `user_item_scores` | `userId, itemId, type, value, createdAt` | the user's scores; it holds every field the aggregation reads, so it is a covered query (no documents fetched) |
| `user_createdAt_id` | `userId, createdAt -1, _id -1` | history; replaces `userId, createdAt -1`, because `_id` is the tie-breaker of the sort and was sorted in memory |

`category + level` was dropped: it is a prefix of `profile_popularity`. Startup drops the two
replaced indexes on existing databases.

Users store only reading preferences, never disability or health information.

## Performance testing

The load test calls `GET /users/me/recommendations?limit=10` while the data grows.

| Size | Users | Items | Interactions |
|---|---|---|---|
| S | 200 | 500 | 10,000 |
| M | 1,000 | 1,000 | 100,000 |
| L | 5,000 | 2,000 | 500,000 |
| XL | 10,000 | 5,000 | 1,000,000 |

**k6 script** (`scripts/k6/recommendations.js`): setup logs in 50 users (`user1@perf.kz` ...
`user50@perf.kz`), then 20 virtual users call the endpoint for 30 s as random ones of them.
Requests still running at 30 s may finish (up to 120 s), so slow modes are measured too.
RPS is the number of completed requests per second of load (without the login setup).

**Runner** (`scripts/perf/run.sh <modes> [sizes]`, needs bash, Go and k6; set `K6=/path/to/k6`
if k6 is not on `PATH`). **It replaces all data in the database.** For each size it seeds the
data, runs the rebuild job (for precomputed modes), saves the query plans, runs the quality
check, and runs k6 once per mode:

```bash
docker compose up -d --build
scripts/perf/run.sh live S M L XL                  # baseline       (make perf ARGS="live")
scripts/perf/run.sh precomputed,cache S M L XL     # optimized      (make perf ARGS=precomputed,cache)
scripts/perf/compare_explain.sh                    # explain before/after table

# afterwards, restore the demo data:
go run ./cmd/seed --reset && go run ./cmd/recompute
```

Modes: `live` (RECS_MODE=live), `precomputed` (CACHE_TTL=0) and `cache` (precomputed with
CACHE_TTL=60s, written as `precomputed+cache`).

| Output | Content |
|---|---|
| `perf/results.csv` | `size, mode, p50, p95, avg, rps` (ms), one row per size and mode |
| `perf/rebuild.csv` | `size, items, duration_ms` of the rebuild job |
| `perf/quality.csv` | `size, method, users, precision@10, recall@10` |
| `perf/explain/<label>/<size>/` | `explain("executionStats")` of 9 main queries as JSON, plus `summary.csv` (plan, index, keys and documents examined). `before` = baseline before the index review, `after` = with the new indexes (label set with `EXPLAIN_LABEL`) |
| `perf/explain/before_after.csv` | the two summaries side by side |

The quality check (`go run ./cmd/evaluate --size demo`) hides the last 20% of each user's liked
items, computes similarities and popularity from the rest, and measures how many hidden items
are found in the top 10 of collaborative filtering (filled with popular items when it has
fewer than 10) and of a popularity baseline.

## Results

Measured on one Windows machine (Docker Desktop, WSL 2), API and MongoDB in Docker, k6 on the host.

**Latency and throughput** (ms; 20 virtual users)

| Size | Mode | p50 | p95 | avg | RPS |
|---|---|---|---|---|---|
| S | live | 1,831 | 2,492 | 1,810 | 10.9 |
| S | precomputed | 44 | 69 | 45 | 439 |
| S | precomputed+cache | 3.6 | 8.4 | 4.1 | 4,358 |
| M | live | 15,829 | 19,477 | 15,426 | 1.2 |
| M | precomputed | 50 | 79 | 51 | 389 |
| M | precomputed+cache | 3.7 | 8.8 | 4.2 | 4,244 |
| L | live | 61,518 | 63,971 | 60,254 | 0.3 |
| L | precomputed | 51 | 81 | 53 | 378 |
| L | precomputed+cache | 3.7 | 8.5 | 4.1 | 4,332 |
| XL | live | 91,784 | 94,747 | 87,530 | 0.2 |
| XL | precomputed | 50 | 79 | 52 | 385 |
| XL | precomputed+cache | 3.6 | 8.3 | 4.1 | 4,409 |

- **live** gets slower roughly in proportion to the data: on XL one request reads the whole
  `interactions` collection for the norms of the co-liked items. From L on, every request
  takes longer than the 30 s test, so each virtual user completes only one request.
- **precomputed** stays at about 50 ms from S to XL: a request reads one user's scores (about
  120 index keys), 50 similarity documents and a few items, whatever the size. On XL it is
  about 1,800 times faster than live at p50.
- **cache** answers in about 4 ms. This is a best case, see Limitations.

**Rebuild job** (the price of precomputing): S 0.12 s, M 0.67 s, L 3.2 s, XL 8.0 s
(5,000 items, 1,000,000 interactions).

**Indexes** (XL, `perf/explain/before_after.csv`):

| Query | Before | After |
|---|---|---|
| popular items | IXSCAN `_id` + in-memory SORT, 4,947 docs | IXSCAN `popularity`, 10 docs |
| items matching the profile | IXSCAN `category_level` + in-memory SORT, 836 docs | IXSCAN `profile_popularity` + SORT_MERGE, 10 docs |
| history page | IXSCAN + in-memory SORT, 122 docs | IXSCAN `user_createdAt_id`, 20 docs |
| user scores | IXSCAN + FETCH, 122 docs | covered IXSCAN `user_item_scores`, 0 docs |
| live co-occurrence | 444,384 docs fetched | 12,599 docs fetched (its `$lookup` is covered too) |
| rebuild scores | COLLSCAN, 1,000,000 docs | unchanged: the job reads every interaction by design |

**Quality** (precision@10 / recall@10):

| Data | Users | Collaborative filtering | Popularity baseline |
|---|---|---|---|
| demo | 50 | 0.070 / 0.169 | 0.012 / 0.029 |
| S | 200 | 0.026 / 0.052 | 0.011 / 0.021 |
| M | 994 | 0.041 / 0.045 | 0.017 / 0.020 |
| L | 4,953 | 0.018 / 0.019 | 0.007 / 0.008 |
| XL | 9,907 | 0.007 / 0.007 | 0.003 / 0.003 |

Collaborative filtering finds 2.3 to 2.6 times (S to XL) and 5.8 times (demo) more of the hidden
items than the popularity baseline.

## Limitations

- **Cache race.** If a recommendation request is still being computed when the same user
  records an interaction, the request can store the old list after the cache entry was
  dropped. The user then sees a stale list for up to `CACHE_TTL`. A per-user version number
  would fix it; it was left out to keep the cache simple.
- **The cache test is a best case.** k6 uses only 50 users and records no interactions, so
  after the first 50 requests almost every request is a cache hit. With many users or frequent
  interactions (opening a text records a `view`, which drops the entry) the hit rate and the
  gain are lower. The cache is in the memory of one API process: several API instances would
  each have their own.
- **Quality numbers are low because the data is synthetic.** The seed picks items only by
  profile weights (interests, language, level), with no real taste or series patterns, and
  with up to 5,000 items a random hidden item is hard to guess. The numbers are useful to
  compare the methods with each other, not as an estimate of real-world quality.
- **Similarities are only as fresh as the last rebuild.** The rebuild recomputes everything
  (8 s on XL) and runs synchronously in the HTTP request of the admin endpoint. It is not
  scheduled; run it after bulk changes, or restart Docker Compose.
- The live baseline is slow by design. It computes on request what the rebuild precomputes,
  and exists to measure that difference.

## Tests

```bash
go test ./...                        # make test
```

Unit tests cover the interaction weights and scores, the similarity formula, the top-20 cut,
the recommendation scoring, the equality of the live and precomputed neighbors, and the TTL
cache.

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
  seed/                 demo and big data generator
  recompute/            rebuild job for item_similarities
  evaluate/             offline quality check (precision@10, recall@10)
  explain/              explain("executionStats") of the main queries
internal/
  config/               environment variables and .env loading
  domain/               User, Item, Interaction, allowed values, errors
  auth/                 bcrypt passwords and JWT tokens
  recommend/            the algorithm: scores, similarities, ranking (no database code)
  cache/                in-memory TTL cache
  repository/mongo/     MongoDB connection, indexes, queries and aggregations
  service/              business logic and validation
  http/
    router.go           routes
    handlers/           HTTP handlers (with Swagger annotations)
    middleware/         authentication, role check and CORS
docs/                   generated Swagger spec
web/                    React web UI (Vite), its Dockerfile and nginx.conf
scripts/
  k6/                   k6 load test
  perf/                 run.sh (all sizes and modes), compare_explain.sh
perf/                   results: results.csv, rebuild.csv, quality.csv, explain/
docker-compose.yml      mongo, mongo-express, init, api, web
Dockerfile              multi-stage build of the API, the seed and the rebuild job
Makefile                shortcuts: up, down, run, seed, recompute, evaluate, perf, swag, test
.env.example            example configuration
```
