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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestResultJSON checks both encodings of Result and the decode failure.
func TestResultJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		give     string
		want     Result[[]int]
		wantJSON string
		wantErr  bool
	}{
		"value":          {give: `[3,7]`, want: Result[[]int]{Value: []int{3, 7}}, wantJSON: `[3,7]`},
		"empty_value":    {give: `[]`, want: Result[[]int]{Value: []int{}}, wantJSON: `[]`},
		"error":          {give: `{"err":"E_CELL_INVALID"}`, want: Result[[]int]{Err: "E_CELL_INVALID"}, wantJSON: `{"err":"E_CELL_INVALID"}`},
		"object_no_err":  {give: `{"other":1}`, wantErr: true},
		"malformed":      {give: `[1,`, wantErr: true},
		"wrong_type":     {give: `"abc"`, wantErr: true},
		"empty_err_name": {give: `{"err":""}`, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got Result[[]int]

			err := json.Unmarshal([]byte(tt.give), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal(%s): err %v, wantErr %v", tt.give, err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if got.Err != tt.want.Err || !slices.Equal(got.Value, tt.want.Value) {
				t.Fatalf("Unmarshal(%s) = %+v, want %+v", tt.give, got, tt.want)
			}

			data, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}

			if string(data) != tt.wantJSON {
				t.Fatalf("Marshal = %s, want %s", data, tt.wantJSON)
			}
		})
	}
}

// TestErrorName covers nil, every defined error, a wrapped error and an
// unknown error.
func TestErrorName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		give error
		want string
	}{
		"nil":         {give: nil, want: ""},
		"cell":        {give: h3go.ErrCellInvalid, want: "E_CELL_INVALID"},
		"wrapped":     {give: fmt.Errorf("outer: %w", h3go.ErrDeletedDigit), want: "E_DELETED_DIGIT"},
		"unknown":     {give: errors.New("something else"), want: "E_FAILED"},
		"failed":      {give: h3go.ErrFailed, want: "E_FAILED"},
		"res_domain":  {give: h3go.ErrResolutionDomain, want: "E_RES_DOMAIN"},
		"base_domain": {give: h3go.ErrBaseCellDomain, want: "E_BASE_CELL_DOMAIN"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := ErrorName(tt.give); got != tt.want {
				t.Fatalf("ErrorName(%v) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}

	t.Run("every_defined_error_has_a_distinct_name", func(t *testing.T) {
		t.Parallel()

		seen := map[string]bool{}
		for err, name := range errorNames {
			if seen[name] {
				t.Errorf("name %s used twice", name)
			}

			seen[name] = true

			if got := ErrorName(err); got != name {
				t.Errorf("ErrorName(%v) = %q, want %q", err, got, name)
			}
		}
	})
}

// TestRecords covers decoding, early termination, a malformed line and a
// failing reader.
func TestRecords(t *testing.T) {
	t.Parallel()

	type point struct {
		X int `json:"x"`
	}

	t.Run("decodes_every_line", func(t *testing.T) {
		t.Parallel()

		var got []int

		for record, err := range Records[point](strings.NewReader("{\"x\":1}\n{\"x\":2}\n")) {
			if err != nil {
				t.Fatal(err)
			}

			got = append(got, record.X)
		}

		if want := []int{1, 2}; !slices.Equal(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("stops_when_the_consumer_breaks", func(t *testing.T) {
		t.Parallel()

		count := 0

		for _, err := range Records[point](strings.NewReader("{\"x\":1}\n{\"x\":2}\n")) {
			if err != nil {
				t.Fatal(err)
			}

			count++

			break
		}

		if count != 1 {
			t.Fatalf("saw %d records, want 1", count)
		}
	})

	t.Run("reports_the_line_of_a_malformed_record", func(t *testing.T) {
		t.Parallel()

		var got error

		for _, err := range Records[point](strings.NewReader("{\"x\":1}\n{\"x\":\n{\"x\":3}\n")) {
			got = err
		}

		if got == nil || !strings.HasPrefix(got.Error(), "line 2:") {
			t.Fatalf("error %v, want a line 2 decode error", got)
		}
	})

	t.Run("reports_a_reader_failure", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("disk on fire")

		var got error

		for _, err := range Records[point](iotest.ErrReader(wantErr)) {
			got = err
		}

		if !errors.Is(got, wantErr) {
			t.Fatalf("error %v, want %v", got, wantErr)
		}
	})
}

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("closed")
}

// TestWriteRecords checks the line format and propagates writer errors.
func TestWriteRecords(t *testing.T) {
	t.Parallel()

	t.Run("one_terminated_line_per_record", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder

		err := WriteRecords(&out, []Result[string]{{Value: "8001fffffffffff"}, {Err: "E_FAILED"}})
		if err != nil {
			t.Fatal(err)
		}

		if want := "\"8001fffffffffff\"\n{\"err\":\"E_FAILED\"}\n"; out.String() != want {
			t.Fatalf("wrote %q, want %q", out.String(), want)
		}
	})

	t.Run("writer_error", func(t *testing.T) {
		t.Parallel()

		if err := WriteRecords(failingWriter{}, []int{1}); err == nil {
			t.Fatal("expected an error")
		}
	})
}

// writeSuite writes a small suite into a temp dir and returns the dir.
func writeSuite(t *testing.T, manifestBody string, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if manifestBody != "" {
		if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifestBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

// TestLoadManifest covers the happy path and every rejection.
func TestLoadManifest(t *testing.T) {
	t.Parallel()

	const body = "{\"index\":\"0\"}\n{\"index\":\"1\"}\n"

	info, err := DescribeFile(writeSuite(t, "", map[string]string{"f.jsonl": body}) + "/f.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	good := fmt.Sprintf(`{"format":%q,"h3Version":"4.5.0","generator":{"name":"g","seed":1},"tolerances":{"angularDeg":1e-11,"relative":1e-9},"files":{"g/f.jsonl":{"records":%d,"sha256":%q}}}`, FormatVersion, info.Records, info.SHA256)

	tests := map[string]struct {
		giveManifest string
		giveFiles    map[string]string
		wantErr      string
	}{
		"valid":            {giveManifest: good, giveFiles: map[string]string{"g/f.jsonl": body}},
		"missing_manifest": {giveManifest: "", wantErr: "no such file"},
		"malformed":        {giveManifest: "{", wantErr: "unexpected end"},
		"wrong_format":     {giveManifest: strings.Replace(good, FormatVersion, "h3-conformance/0", 1), wantErr: "format"},
		"missing_file":     {giveManifest: good, wantErr: "no such file"},
		"changed_file":     {giveManifest: good, giveFiles: map[string]string{"g/f.jsonl": body + "{}\n"}, wantErr: "manifest says"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := writeSuite(t, tt.giveManifest, tt.giveFiles)

			manifest, err := LoadManifest(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("LoadManifest: err %v, want it to contain %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if manifest.H3Version != "4.5.0" || manifest.Files["g/f.jsonl"] != info {
				t.Fatalf("manifest %+v", manifest)
			}
		})
	}
}

// TestWriteManifest round-trips a manifest through disk and propagates a
// write failure.
func TestWriteManifest(t *testing.T) {
	t.Parallel()

	t.Run("round_trip", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		want := Manifest{
			Format:     FormatVersion,
			H3Version:  "4.5.0",
			Generator:  Generator{Name: "g", Seed: 7},
			Tolerances: Tolerances{AngularDeg: 1e-11, Relative: 1e-9},
			Files:      map[string]FileInfo{},
		}

		if err := WriteManifest(dir, want); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(filepath.Join(dir, ManifestFile))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.HasSuffix(string(data), "}\n") || !strings.Contains(string(data), "\n  \"format\"") {
			t.Fatalf("manifest is not indented and newline-terminated:\n%s", data)
		}

		got, err := LoadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}

		if got.Generator != want.Generator || got.Tolerances != want.Tolerances || got.H3Version != want.H3Version {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("unwritable_dir", func(t *testing.T) {
		t.Parallel()

		if err := WriteManifest(filepath.Join(t.TempDir(), "missing"), Manifest{}); err == nil {
			t.Fatal("expected an error")
		}
	})
}

// TestDescribeFile checks the count and hash of a known file and the missing
// file error.
func TestDescribeFile(t *testing.T) {
	t.Parallel()

	t.Run("known_content", func(t *testing.T) {
		t.Parallel()

		dir := writeSuite(t, "", map[string]string{"f.jsonl": "a\nb\n"})

		got, err := DescribeFile(filepath.Join(dir, "f.jsonl"))
		if err != nil {
			t.Fatal(err)
		}

		want := FileInfo{Records: 2, SHA256: "911169ddaaf146aff539f58c26c489af3b892dff0fe283c1c264c65ae5aa59a2"}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		t.Parallel()

		if _, err := DescribeFile(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Fatal("expected an error")
		}
	})
}
