#!/usr/bin/env bash
# Runs the recommendations load test for each data size and appends one row
# per size and mode to perf/results.csv (size,mode,p50,p95,avg,rps; ms).
#
#   scripts/perf/run.sh <modes> [sizes...]
#   scripts/perf/run.sh live                    # baseline, all sizes
#   scripts/perf/run.sh precomputed,cache       # optimized, all sizes
#   scripts/perf/run.sh live S M                # only S and M
#
# modes (comma-separated):
#   live         RECS_MODE=live, no cache             -> row mode "live"
#   precomputed  RECS_MODE=precomputed, CACHE_TTL=0   -> row mode "precomputed"
#   cache        RECS_MODE=precomputed, CACHE_TTL=60s -> row mode "precomputed+cache"
# sizes: S M L XL (default all).
#
# For each size the script:
#   1. seeds the data (fixed random seed, so the data is the same every time),
#   2. runs the rebuild job if a precomputed mode is tested and appends its
#      duration to perf/rebuild.csv,
#   3. saves explain("executionStats") of the main queries to
#      perf/explain/$EXPLAIN_LABEL/<size> (default label "after"),
#   4. appends precision@10 / recall@10 to perf/quality.csv (cmd/evaluate),
#   5. runs k6 once per mode.
#
# Needs: Docker Compose stack running (mongo, api), Go, k6 (set K6=/path/to/k6
# if it is not on PATH). WARNING: the seed replaces all data in the database.
set -euo pipefail
cd "$(dirname "$0")/../.."

MODES=${1:?usage: scripts/perf/run.sh <modes> [sizes...]}
shift
SIZES=${*:-S M L XL}
K6=${K6:-k6}
API=${API:-http://localhost:8080}
EXPLAIN_LABEL=${EXPLAIN_LABEL:-after}
RESULTS=perf/results.csv
ROW=perf/.row.csv

seed_args() {
  case $1 in
    S) echo "--users 200 --items 500 --interactions 10000" ;;
    M) echo "--users 1000 --items 1000 --interactions 100000" ;;
    L) echo "--users 5000 --items 2000 --interactions 500000" ;;
    XL) echo "--users 10000 --items 5000 --interactions 1000000" ;;
    *) echo "unknown size $1" >&2; exit 1 ;;
  esac
}

# start_api <RECS_MODE> <CACHE_TTL>: recreate the api container with these settings and wait for it.
start_api() {
  # --no-deps: do not run the init job (seed + rebuild) again.
  RECS_MODE=$1 CACHE_TTL=$2 docker compose up -d --no-deps api >/dev/null 2>&1
  until curl -sf "$API/health" >/dev/null; do sleep 1; done
}

mkdir -p perf
[ -f "$RESULTS" ] || echo "size,mode,p50,p95,avg,rps" > "$RESULTS"

for size in $SIZES; do
  echo "=== size $size: seeding ($(seed_args "$size"))"
  go run ./cmd/seed --reset $(seed_args "$size")

  if [[ ",$MODES," == *",precomputed,"* || ",$MODES," == *",cache,"* ]]; then
    echo "--- $size: rebuild job"
    go run ./cmd/recompute --size "$size" --results perf/rebuild.csv
  fi

  # The API creates the indexes at startup, so start it before explaining.
  start_api precomputed 0
  echo "--- $size: explain -> perf/explain/$EXPLAIN_LABEL/$size"
  go run ./cmd/explain --out "perf/explain/$EXPLAIN_LABEL/$size"

  echo "--- $size: quality (precision@10, recall@10)"
  go run ./cmd/evaluate --size "$size" --out perf/quality.csv

  for mode in ${MODES//,/ }; do
    case $mode in
      live) start_api live 0; label=live ;;
      precomputed) start_api precomputed 0; label=precomputed ;;
      cache) start_api precomputed 60s; label=precomputed+cache ;;
      *) echo "unknown mode $mode" >&2; exit 1 ;;
    esac

    echo "--- $size / $label: k6"
    "$K6" run -q -e SIZE="$size" -e MODE="$label" -e ROW_FILE="$ROW" -e BASE_URL="$API" \
      scripts/k6/recommendations.js
    cat "$ROW" >> "$RESULTS"
    rm -f "$ROW"
  done
done

echo "=== $RESULTS"
cat "$RESULTS"
