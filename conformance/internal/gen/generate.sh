#!/bin/sh
# Builds gen.c against the H3 sources pinned in third_party/h3 (see build.sh)
# and writes the suite into the directory given as the first argument
# (default: testdata, relative to the current directory). Requires a C99
# compiler.
set -eu

out=${1:-testdata}
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../.." && pwd)
build=$(mktemp -d)
trap 'rm -rf "$build"' EXIT

sh "$here/build.sh" "$build/gen"

mkdir -p "$out/inspection" "$out/hierarchy" "$out/traversal" "$out/unsafe" "$out/edges" "$out/vertexes" "$out/localij" "$out/sets" "$out/regions" "$out/digests" "$out/floats"
"$build/gen" -o "$out" -v "$(cat "$root/H3_VERSION")" -s "${H3_CONFORMANCE_SEED:-20260929}"
