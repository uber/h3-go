/*
 * Copyright 2026 Uber Technologies, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package conformance defines the file format of the H3 conformance suite and
// runs the suite against the pure-Go h3go package.
//
// The suite is a set of newline-delimited JSON files under testdata, each line
// describing one input and the outputs the H3 reference implementation
// produces for it, plus a manifest that pins the H3 version and the file
// hashes. README.md in this directory is the format specification; it is
// written to be implemented from any language without reference to this
// package.
//
// The files are produced by internal/gen/gen.c, a standalone C program built
// against the vendored H3 sources so that it can move to the H3 repository
// unchanged. Regenerate them with go generate after changing the generator or
// the vendored H3 version.
package conformance

//go:generate sh internal/gen/generate.sh testdata
