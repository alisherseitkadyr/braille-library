#!/usr/bin/env bash
# Joins perf/explain/before/<size>/summary.csv and perf/explain/after/<size>/summary.csv
# into perf/explain/before_after.csv: one row per size and query with the plan,
# keys and documents examined before and after the index review.
set -euo pipefail
cd "$(dirname "$0")/../.."

OUT=perf/explain/before_after.csv
echo "size,query,before_plan,before_keys,before_docs,after_plan,after_keys,after_docs" > "$OUT"
for size in S M L XL; do
  before=perf/explain/before/$size/summary.csv
  after=perf/explain/after/$size/summary.csv
  [ -f "$before" ] && [ -f "$after" ] || continue
  # summary.csv: query,collection,time_ms,plan,indexes,keys_examined,docs_examined,returned
  awk -F, -v size="$size" '
    NR == FNR { if (FNR > 1) { plan[$1] = $4 " (" $5 ")"; keys[$1] = $6; docs[$1] = $7 }; next }
    FNR > 1 && ($1 in plan) {
      print size "," $1 "," plan[$1] "," keys[$1] "," docs[$1] "," $4 " (" $5 ")," $6 "," $7
    }' "$before" "$after" >> "$OUT"
done
column -s, -t "$OUT" 2>/dev/null || cat "$OUT"
