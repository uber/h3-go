#!/bin/sh
# Compiles gen.c against the H3 sources pinned in third_party/h3 and writes
# the binary to the path given as the first argument. Requires a C99
# compiler and the submodule to be checked out.
#
#   build.sh <output-binary>
#
# The sources are copied into a scratch tree so the submodule is never
# modified: h3api.h is rendered from the CMake template, every file in
# ./patches is applied, and the result is compiled. A patch that no longer
# applies, because the pinned release contains it, fails the build; delete
# the patch then. GEN_CFLAGS adds compiler flags to the whole build. When
# GEN_STRICT_CFLAGS is set, gen.c alone is also checked with those flags
# (syntax only), so the generator can be held to warnings the library is not.
set -eu

out=$1
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../.." && pwd)
h3="$root/third_party/h3"
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

if [ ! -f "$h3/src/h3lib/include/h3api.h.in" ]; then
    echo "build.sh: $h3 is empty; run: git submodule update --init third_party/h3" >&2
    exit 1
fi

pinned=$(cat "$root/H3_VERSION")
upstream=$(cat "$h3/VERSION")
upstream=${upstream%%-*}
if [ "$pinned" != "v$upstream" ]; then
    echo "build.sh: H3_VERSION is $pinned but third_party/h3 is v$upstream" >&2
    exit 1
fi

tree="$scratch/h3"
mkdir -p "$tree/src/h3lib/lib" "$tree/src/h3lib/include"
cp "$h3"/src/h3lib/lib/*.c "$tree/src/h3lib/lib/"
cp "$h3"/src/h3lib/include/*.h "$tree/src/h3lib/include/"

major=${upstream%%.*}
rest=${upstream#*.}
minor=${rest%%.*}
patchlevel=${rest#*.}
sed -e "s/@H3_VERSION_MAJOR@/$major/" -e "s/@H3_VERSION_MINOR@/$minor/" \
    -e "s/@H3_VERSION_PATCH@/$patchlevel/" \
    "$h3/src/h3lib/include/h3api.h.in" > "$tree/src/h3lib/include/h3api.h"

for file in "$here"/patches/*.patch; do
    [ -e "$file" ] || continue
    patch -p1 -N -s -d "$tree" < "$file"
done

if [ -n "${GEN_STRICT_CFLAGS:-}" ]; then
    # shellcheck disable=SC2086 # GEN_STRICT_CFLAGS is a list of flags
    ${CC:-cc} -std=c99 -fsyntax-only $GEN_STRICT_CFLAGS \
        -I"$tree/src/h3lib/include" "$here/gen.c"
fi

# shellcheck disable=SC2086 # GEN_CFLAGS is a list of flags
${CC:-cc} -std=c99 -O2 -DH3_HAVE_VLA=1 ${GEN_CFLAGS:-} \
    -I"$tree/src/h3lib/include" -o "$out" "$here/gen.c" "$tree"/src/h3lib/lib/*.c -lm
