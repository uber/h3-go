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
	"math"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// earthRadiusKm is the sphere radius the reference uses for kilometre and
// metre outputs, as README.md states for the absolute tolerance floor.
const earthRadiusKm = 6371.007180918475

// unit scales a measure from radians (or steradians) to its output unit, and
// says whether the measure is a length or an area.
type unit struct {
	scale float64
	area  bool
}

// The units the floats group uses.
var (
	rads  = unit{scale: 1}                               //nolint:gochecknoglobals // fixed by the format
	km    = unit{scale: earthRadiusKm}                   //nolint:gochecknoglobals // fixed by the format
	m     = unit{scale: earthRadiusKm * 1000}            //nolint:gochecknoglobals // fixed by the format
	rads2 = unit{scale: 1, area: true}                   //nolint:gochecknoglobals // fixed by the format
	km2   = unit{scale: earthRadiusKm, area: true}       //nolint:gochecknoglobals // fixed by the format
	m2    = unit{scale: earthRadiusKm * 1e3, area: true} //nolint:gochecknoglobals // fixed by the format
)

// measurement is a float-returning call's outcome.
type measurement struct {
	value float64
	err   error
}

// measure packs a (float64, error) return into a measurement, so a call can
// be passed straight to the check helpers.
func measure(value float64, err error) measurement {
	return measurement{value: value, err: err}
}

// TestFloatCells runs every floats/cells record against h3go.
func TestFloatCells(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "floats/cells", func(t *testing.T, record FloatCellRecord) {
		t.Helper()

		cell := h3go.CellFromString(record.Index)

		center, err := cell.LatLng()
		if err != nil || !withinAngular(center, record.Center, tolerances.AngularDeg) {
			t.Errorf("LatLng() = %v (%v), want %v", center, err, record.Center)
		}

		boundary, err := cell.Boundary()
		if err != nil || !sequenceWithinAngular(boundary, record.Boundary, tolerances.AngularDeg) {
			t.Errorf("Boundary() = %v (%v), want %v", boundary, err, record.Boundary)
		}

		checkMeasure(t, "CellAreaRads2", record.AreaRads2, rads2, tolerances, measure(h3go.CellAreaRads2(cell)))
		checkMeasure(t, "CellAreaKm2", record.AreaKm2, km2, tolerances, measure(h3go.CellAreaKm2(cell)))
		checkMeasure(t, "CellAreaM2", record.AreaM2, m2, tolerances, measure(h3go.CellAreaM2(cell)))

		if len(record.Interior) != len(record.InteriorCells) {
			t.Fatalf("record has %d interior points and %d cells", len(record.Interior), len(record.InteriorCells))
		}

		for i, point := range record.Interior {
			got, err := h3go.LatLngToCell(h3go.LatLng{Lat: point[0], Lng: point[1]}, record.InteriorRes)
			if err != nil || got.String() != record.InteriorCells[i] {
				t.Errorf("LatLngToCell(%v, %d) = %s (%v), want %s", point, record.InteriorRes, got, err, record.InteriorCells[i])
			}
		}
	})
}

// TestFloatEdges runs every floats/edges record against h3go.
func TestFloatEdges(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "floats/edges", func(t *testing.T, record FloatEdgeRecord) {
		t.Helper()

		edge := h3go.DirectedEdgeFromString(record.Index)

		checkMeasure(t, "EdgeLengthRads", record.LengthRads, rads, tolerances, measure(h3go.EdgeLengthRads(edge)))
		checkMeasure(t, "EdgeLengthKm", record.LengthKm, km, tolerances, measure(h3go.EdgeLengthKm(edge)))
		checkMeasure(t, "EdgeLengthM", record.LengthM, m, tolerances, measure(h3go.EdgeLengthM(edge)))

		boundary, err := edge.Boundary()
		if err != nil || !sequenceWithinAngular(boundary, record.Boundary, tolerances.AngularDeg) {
			t.Errorf("Boundary() = %v (%v), want %v", boundary, err, record.Boundary)
		}
	})
}

// TestFloatVertexes runs every floats/vertexes record against h3go.
func TestFloatVertexes(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "floats/vertexes", func(t *testing.T, record FloatVertexRecord) {
		t.Helper()

		vertex := h3go.VertexFromString(record.Index)

		point, err := vertex.LatLng()
		if err != nil || !withinAngular(point, record.LatLng, tolerances.AngularDeg) {
			t.Errorf("LatLng() = %v (%v), want %v", point, err, record.LatLng)
		}
	})
}

// TestDistances runs every floats/distances record against h3go.
func TestDistances(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "floats/distances", func(t *testing.T, record DistanceRecord) {
		t.Helper()

		a := h3go.LatLng{Lat: record.A[0], Lng: record.A[1]}
		b := h3go.LatLng{Lat: record.B[0], Lng: record.B[1]}

		checkMeasure(t, "GreatCircleDistanceRads", record.Rads, rads, tolerances, measure(h3go.GreatCircleDistanceRads(a, b), nil))
		checkMeasure(t, "GreatCircleDistanceKm", record.Km, km, tolerances, measure(h3go.GreatCircleDistanceKm(a, b), nil))
		checkMeasure(t, "GreatCircleDistanceM", record.M, m, tolerances, measure(h3go.GreatCircleDistanceM(a, b), nil))
	})
}

// TestFloatResolutions runs every floats/resolutions record against h3go.
func TestFloatResolutions(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "floats/resolutions", func(t *testing.T, record FloatResolutionRecord) {
		t.Helper()

		checkMeasureResult(t, "HexagonAreaAvgKm2", record.HexagonAreaKm2, km2, tolerances, measure(h3go.HexagonAreaAvgKm2(record.Res)))
		checkMeasureResult(t, "HexagonAreaAvgM2", record.HexagonAreaM2, m2, tolerances, measure(h3go.HexagonAreaAvgM2(record.Res)))
		checkMeasureResult(t, "HexagonEdgeLengthAvgKm", record.EdgeLengthKm, km, tolerances, measure(h3go.HexagonEdgeLengthAvgKm(record.Res)))
		checkMeasureResult(t, "HexagonEdgeLengthAvgM", record.EdgeLengthM, m, tolerances, measure(h3go.HexagonEdgeLengthAvgM(record.Res)))
	})
}

// checkMeasure asserts a measurement call succeeded and is within tolerance
// of the expected value.
func checkMeasure(t *testing.T, name string, want float64, u unit, tolerances Tolerances, got measurement) {
	t.Helper()

	if got.err != nil || !withinMeasure(got.value, want, u, tolerances) {
		t.Errorf("%s = %v (%v), want %v", name, got.value, got.err, want)
	}
}

// checkMeasureResult asserts a measurement call that may fail matches the
// expected Result: the same error name, or a value within tolerance.
func checkMeasureResult(t *testing.T, name string, want Result[float64], u unit, tolerances Tolerances, got measurement) {
	t.Helper()

	if errName := ErrorName(got.err); errName != want.Err {
		t.Errorf("%s error = %q, want %q", name, errName, want.Err)

		return
	}

	if got.err == nil && !withinMeasure(got.value, want.Value, u, tolerances) {
		t.Errorf("%s = %v, want %v", name, got.value, want.Value)
	}
}

// withinMeasure applies the README's rule for lengths and areas: the
// difference must be within the relative tolerance of the expected value or
// within the absolute floor derived from the angular tolerance.
func withinMeasure(got, want float64, u unit, tolerances Tolerances) bool {
	diff := math.Abs(got - want)
	if diff <= tolerances.Relative*math.Abs(want) {
		return true
	}

	angularRad := tolerances.AngularDeg * math.Pi / 180

	floor := angularRad * u.scale
	if u.area {
		floor = angularRad * math.Sqrt(math.Abs(want)) * u.scale
	}

	return diff <= floor
}

// sequenceWithinAngular reports whether two point sequences have the same
// length and are pairwise within the angular tolerance, in order.
func sequenceWithinAngular(got []h3go.LatLng, want []GeoCoord, angularDeg float64) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range want {
		if !withinAngular(got[i], want[i], angularDeg) {
			return false
		}
	}

	return true
}
