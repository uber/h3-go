#!/bin/sh
# Produces the digest rows of one resolution with one generator process per
# base cell, running JOBS processes at a time (default: every online CPU),
# replaces that resolution's rows in the suite's digest files, and rewrites
# the manifest. generate.sh writes resolutions 0 through 6 in one process;
# this is how resolution 7, about an hour single-threaded, is produced.
#
#   digests.sh <res> [suite-dir]
set -eu

res=$1
out=${2:-testdata}
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../../../.." && pwd)
build=$(mktemp -d)
trap 'rm -rf "$build"' EXIT
jobs=${JOBS:-$(getconf _NPROCESSORS_ONLN)}

mkdir -p "$build/include" "$build/rows"
ln -s "$root/h3_h3api.h" "$build/include/h3api.h"
${CC:-cc} -std=c99 -O2 -DH3_HAVE_VLA=1 -I"$root" -I"$build/include" \
    -o "$build/gen" "$here/gen.c" "$root"/h3_*.c -lm

# One process per base cell; each writes its row to its own file so the rows
# can be concatenated in base-cell order afterwards.
seq 0 121 | xargs -P "$jobs" -I'{}' sh -c \
    '"$1" --digests "$2" "$3" "$3" > "$4/$3.jsonl"' sh "$build/gen" "$res" '{}' "$build/rows"

for baseCell in $(seq 0 121); do
    cat "$build/rows/$baseCell.jsonl"
done > "$build/base.jsonl"
"$build/gen" --resolution-row "$res" < "$build/base.jsonl" > "$build/res.jsonl"

# Replace any existing rows of this resolution, keeping the others in order.
for file in baseCells resolutions; do
    grep -v "^{\"res\":$res," "$out/digests/$file.jsonl" > "$build/$file.keep" || true
done
cat "$build/baseCells.keep" "$build/base.jsonl" > "$out/digests/baseCells.jsonl"
cat "$build/resolutions.keep" "$build/res.jsonl" > "$out/digests/resolutions.jsonl"

"$build/gen" -o "$out" -v "$(cat "$root/H3_VERSION")" -s "${H3_CONFORMANCE_SEED:-20260929}" --manifest
