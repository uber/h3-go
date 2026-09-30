#!/bin/sh
# Builds gen.c against the vendored H3 sources at the repository root and
# writes the suite into the directory given as the first argument (default:
# testdata, relative to the current directory). Requires a C99 compiler.
set -eu

out=${1:-testdata}
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../../../.." && pwd)
build=$(mktemp -d)
trap 'rm -rf "$build"' EXIT

# gen.c includes <h3api.h> as an upstream testapp would; the vendored copy is
# named h3_h3api.h, so expose it under the upstream name.
mkdir -p "$build/include"
ln -s "$root/h3_h3api.h" "$build/include/h3api.h"

${CC:-cc} -std=c99 -O2 -DH3_HAVE_VLA=1 -I"$root" -I"$build/include" \
    -o "$build/gen" "$here/gen.c" "$root"/h3_*.c -lm

mkdir -p "$out/inspection"
"$build/gen" -o "$out" -v "$(cat "$root/H3_VERSION")" -s "${H3_CONFORMANCE_SEED:-20260929}"
