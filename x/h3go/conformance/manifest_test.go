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

package conformance

import (
	"io/fs"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// suiteDir is the checked-in suite the tests run against.
const suiteDir = "testdata"

// vendoredVersionFile holds the version of the C library vendored at the
// repository root, which the suite must have been generated from.
var vendoredVersionFile = filepath.Join("..", "..", "..", "H3_VERSION")

// loadSuite loads and verifies the checked-in manifest.
func loadSuite(t *testing.T) Manifest {
	t.Helper()

	manifest, err := LoadManifest(suiteDir)
	if err != nil {
		t.Fatal(err)
	}

	return manifest
}

// manifestFiles yields the manifest's file names under a group prefix.
func manifestFiles(manifest Manifest, prefix string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for name := range maps.Keys(manifest.Files) {
			if strings.HasPrefix(name, prefix) && !yield(name) {
				return
			}
		}
	}
}

// TestSuiteManifest checks that the checked-in suite is internally consistent
// and matches the vendored C library version.
func TestSuiteManifest(t *testing.T) {
	t.Parallel()

	manifest := loadSuite(t)

	t.Run("h3_version_matches_vendored", func(t *testing.T) {
		t.Parallel()

		data, err := os.ReadFile(vendoredVersionFile)
		if err != nil {
			t.Fatal(err)
		}

		want := strings.TrimPrefix(strings.TrimSpace(string(data)), "v")
		if manifest.H3Version != want {
			t.Fatalf("manifest h3Version %q, vendored %q: regenerate the suite", manifest.H3Version, want)
		}
	})

	t.Run("every_record_file_is_listed", func(t *testing.T) {
		t.Parallel()

		var onDisk []string

		err := filepath.WalkDir(suiteDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !entry.IsDir() && filepath.Ext(path) == ".jsonl" {
				rel, err := filepath.Rel(suiteDir, path)
				if err != nil {
					return err
				}

				onDisk = append(onDisk, filepath.ToSlash(rel))
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		for _, name := range onDisk {
			if _, ok := manifest.Files[name]; !ok {
				t.Errorf("%s is on disk but not in the manifest", name)
			}
		}

		if len(onDisk) != len(manifest.Files) {
			t.Errorf("%d record files on disk, %d in the manifest", len(onDisk), len(manifest.Files))
		}
	})

	t.Run("tolerances_are_set", func(t *testing.T) {
		t.Parallel()

		if manifest.Tolerances.AngularDeg <= 0 || manifest.Tolerances.Relative <= 0 {
			t.Fatalf("tolerances %+v must be positive", manifest.Tolerances)
		}
	})

	t.Run("generator_is_named", func(t *testing.T) {
		t.Parallel()

		if manifest.Generator.Name == "" {
			t.Fatal("manifest has no generator name")
		}
	})
}
