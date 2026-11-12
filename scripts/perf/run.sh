#!/usr/bin/env bash
# Runs the recommendations load test for each data size and appends one row
# per size and mode to perf/results.csv (size,mode,p50,p95,avg,rps; ms).
#
#   scripts/perf/run.sh <modes> [sizes...]
#   scripts/perf/run.sh live                 # baseline, all sizes
#   scripts/perf/run.sh live S M             # only S and M
#
# modes: comma-separated list of live, precomputed.
# sizes: S M L XL (default all). Each size is seeded once (fixed random seed,
# so the data is the same every time) and then tested in every mode.
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

# start_api <RECS_MODE>: recreate the api container with that mode and wait for it.
start_api() {
  RECS_MODE=$1 docker compose up -d api >/dev/null 2>&1
  until curl -sf "$API/health" >/dev/null; do sleep 1; done
}

mkdir -p perf
[ -f "$RESULTS" ] || echo "size,mode,p50,p95,avg,rps" > "$RESULTS"

for size in $SIZES; do
  echo "=== size $size: seeding ($(seed_args "$size"))"
  go run ./cmd/seed --reset $(seed_args "$size")

  for mode in ${MODES//,/ }; do
    case $mode in
      live) start_api live ;;
      precomputed)
        go run ./cmd/recompute
        start_api precomputed
        ;;
      *) echo "unknown mode $mode" >&2; exit 1 ;;
    esac

    echo "--- $size / $mode: explain"
    go run ./cmd/explain --out "perf/explain/$mode/$size"

    echo "--- $size / $mode: k6"
    "$K6" run -q -e SIZE="$size" -e MODE="$mode" -e ROW_FILE="$ROW" -e BASE_URL="$API" \
      scripts/k6/recommendations.js
    cat "$ROW" >> "$RESULTS"
    rm -f "$ROW"
  done
done

echo "=== $RESULTS"
cat "$RESULTS"
