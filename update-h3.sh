#!/usr/bin/env bash
#
# Copyright 2018 Uber Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not
# use this file except in compliance with the License. You may obtain a copy of
# the License at
#
#         http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations under
# the License.
#

# Arguments: [git-remote]
#
# git-remote - The git remote to pull from. An existing cloned repository will
#              not be deleted if a new remote is specified. Defaults to
#              "github.com/uber/h3"
#
# Will fetch the version of H3 specified in the file `H3_VERSION`, copy the
# source files into the working directory with `h3_` prefix, and headers files
# into `H3_INC_DIR`.

# -- quiet pushd/popd ---
pushd () {
    command pushd "$@" > /dev/null
}

popd () {
    command popd "$@" > /dev/null
}
# -- -- -- -- -- -- -- --

cleanup () {
    echo "Cleaning up!"
    rm -rf "$H3_SRC_DIR"
}
trap cleanup EXIT

GIT_REMOTE=${1:-"https://github.com/uber/h3.git"}
H3_SRC_DIR="src"

# this must match the CGO include path in main.go
H3_INC_DIR="include"

# hold onto the current working directory to copy source files into.
CWD=$(pwd)

# clean up existing C source code.
find . -name "*.c" -depth 1 -exec rm {} \;
if [ -d "$H3_INC_DIR" ]; then
    rm -rf "$H3_INC_DIR"
fi

echo Downloading H3 from "$GIT_REMOTE"

if  [ -d "$H3_SRC_DIR" ]; then
    echo Replacing existing src at "$H3_SRC_DIR"
    rm -rf "$H3_SRC_DIR"
fi

H3_VERSION=$(cat H3_VERSION)
echo "Checking out $H3_VERSION (found in file H3_VERSION)"

git clone "$GIT_REMOTE" "$H3_SRC_DIR"

pushd "$H3_SRC_DIR"
    git checkout -q tags/"$H3_VERSION"

    echo Copying source files into working directory
    pushd ./src/h3lib/lib/
        for f in *.c; do
            cp -- "$f" "$CWD/h3_$f" 2>&1 > /dev/null
        done
    popd
    cp -R ./src/h3lib/include/. "$CWD"/include
popd
