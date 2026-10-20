#!/usr/bin/env bash
# Pins a new H3 release for the conformance suite.
#
#   update-h3.sh <tag>        for example: update-h3.sh v4.6.0
#
# Moves the third_party/h3 submodule to the tag, writes H3_VERSION, updates
# the README badge and prints the regeneration steps that must follow. The
# library itself is pure Go and does not use the submodule.
set -euo pipefail

tag=${1:?usage: update-h3.sh <tag>}
root=$(cd "$(dirname "$0")" && pwd)
h3="$root/third_party/h3"

if [ ! -f "$h3/VERSION" ]; then
    git -C "$root" submodule update --init third_party/h3
fi

git -C "$h3" fetch --tags origin
git -C "$h3" checkout -q "tags/$tag"
printf '%s\n' "$tag" > "$root/H3_VERSION"

sed -i.bak -E \
    "s|/badge/h3-v[^-]+-blue\.svg|/badge/h3-$tag-blue.svg|; s|/uber/h3/releases/tag/v[^)]+|/uber/h3/releases/tag/$tag|" \
    "$root/README.md"
rm "$root/README.md.bak"

cat <<EOF
Pinned $tag. Next:
  1. Review conformance/internal/gen/patches/ and delete any patch the release
     contains; build.sh fails on a patch that is already applied.
  2. go generate ./conformance
  3. sh conformance/internal/gen/digests.sh 7 conformance/testdata
  4. go test ./... and review the testdata diff.
  5. git add .gitmodules third_party/h3 H3_VERSION README.md conformance
EOF
