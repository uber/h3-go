/*
 * Copyright 2018 Uber Technologies, Inc.
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
package h3

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const eps = 1e-4
const validH3Index = H3Index(0x850dab63fffffff)
const pentagonH3Index = H3Index(0x821c07fffffffff)

var (
	validH3Rings1 = [][]H3Index{
		{
			validH3Index,
		},
		{
			0x850dab73fffffff,
			0x850dab7bfffffff,
			0x850dab6bfffffff,
			0x850dab6ffffffff,
			0x850dab67fffffff,
			0x850dab77fffffff,
		},
		{
			0x850dab0bfffffff,
			0x850dab47fffffff,
			0x850dab4ffffffff,
			0x850d8cb7fffffff,
			0x850d8ca7fffffff,
			0x850d8dd3fffffff,
			0x850d8dd7fffffff,
			0x850d8d9bfffffff,
			0x850d8d93fffffff,
			0x850dab2bfffffff,
			0x850dab3bfffffff,
			0x850dab0ffffffff,
		},
	}
	validH3Rings2 = [][]H3Index{
		{
			0x8928308280fffff,
		}, {
			0x8928308280bffff,
			0x89283082873ffff,
			0x89283082877ffff,
			0x8928308283bffff,
			0x89283082807ffff,
			0x89283082803ffff,
		},
		{
			0x8928308281bffff,
			0x89283082857ffff,
			0x89283082847ffff,
			0x8928308287bffff,
			0x89283082863ffff,
			0x89283082867ffff,
			0x8928308282bffff,
			0x89283082823ffff,
			0x89283082833ffff,
			0x892830828abffff,
			0x89283082817ffff,
			0x89283082813ffff,
		},
	}

	validGeoCoord = GeoCoord{
		Latitude:  67.15092686397713,
		Longitude: 191.6091114190303,
	}

	validGeofence = GeoBoundary{
		{Latitude: 67.224749856, Longitude: 191.476993415},
		{Latitude: 67.140938355, Longitude: 191.373085667},
		{Latitude: 67.067252558, Longitude: 191.505086715},
		{Latitude: 67.077062918, Longitude: 191.740304069},
		{Latitude: 67.160561948, Longitude: 191.845198829},
		{Latitude: 67.234563187, Longitude: 191.713897218},
	}

	validGeoRing = []GeoCoord{{}}

	validPolyfillFences []*Geofence = []*Geofence{
		{
			GeoCoords: &GeoCoords{
				Coords: []*GeoCoord{
					{
						19.953317642211914,
						52.90636558063081,
					},
					{
						19.980440139770508,
						52.90636558063081,
					},
					{
						19.980440139770508,
						52.92282408926116,
					},
					{
						19.953317642211914,
						52.92282408926116,
					},
					{
						19.953317642211914,
						52.90636558063081,
					},
				},
			},
		},

		{
			GeoCoords: &GeoCoords{
				Coords: []*GeoCoord{
					{
						21.08696937561035,
						52.17041802836794,
					},
					{
						21.08767747879028,
						52.17268154422601,
					},
					{
						21.083557605743408,
						52.17477387796151,
					},
					{
						21.075360774993896,
						52.17202355730302,
					},
					{
						21.07673406600952,
						52.16502197378441,
					},
					{
						21.086883544921875,
						52.16439019777352,
					},
					{
						21.091046333312985,
						52.16823336315624,
					},
					{
						21.08475923538208,
						52.167785888371064,
					},
					{
						21.08349323272705,
						52.169075656426074,
					},
					{
						21.08696937561035,
						52.17041802836794,
					},
				},
			},
		},
		{
			GeoCoords: &GeoCoords{
				Coords: []*GeoCoord{
					{
						21.085832118988037,
						52.17206797172662,
					},
					{
						21.08570337295532,
						52.172204504677005,
					},
					{
						21.085188388824463,
						52.17201862236433,
					},
					{
						21.08533054590225,
						52.171877153889106,
					},
					{
						21.085832118988037,
						52.17206797172662,
					},
				},
			},
		},
	}
	validPolyfillHoles = [][]*Geofence{
		{},
		{},
		{&Geofence{
			GeoCoords: &GeoCoords{
				Coords: []*GeoCoord{
					{
						21.08563631772995,
						52.17211238610587,
					},
					{
						21.08567386865616,
						52.17206632674875,
					},
					{
						21.085743606090542,
						52.17208442150187,
					},
					{
						21.08570337295532,
						52.17212225596189,
					},
					{
						21.085655093193054,
						52.17211074112963,
					},
					{
						21.08563631772995,
						52.17211238610587,
					},
				},
			},
		}},
	}

	validH3LinkedGeoPolygonSingle = []H3Index{
		0x8d526520b612b3f,
	}
	validH3LinkedGeoPolygons = []H3Index{
		0x8928308280bffff,
		0x89283082873ffff,
		0x89283082877ffff,
		0x8928308283bffff,
		0x89283082807ffff,
		0x89283082803ffff,

		0x8d526520b612b3f,
	}
)

func TestFromGeo(t *testing.T) {
	t.Parallel()
	h := FromGeo(GeoCoord{
		Latitude:  67.194013596,
		Longitude: 191.598258018,
	}, 5)
	assert.Equal(t, validH3Index, h)
}

func TestToGeo(t *testing.T) {
	t.Parallel()
	g := ToGeo(validH3Index)
	assertGeoCoord(t, validGeoCoord, g)
}

func TestToGeoBoundary(t *testing.T) {
	t.Parallel()
	boundary := ToGeoBoundary(validH3Index)
	assertGeoCoords(t, validGeofence[:], boundary[:])
}

func TestSetToLinkedGeoPolygon(t *testing.T) {
	t.Parallel()
	t.Run("single", func(t *testing.T) {
		var h3s []H3Index = []H3Index{validH3LinkedGeoPolygonSingle[0]}
		linkedGP := SetToLinkedGeo(h3s)
		if assert.NotNil(t, linkedGP) {
			defer linkedGP.Destroy()

			assert.Nil(t, linkedGP.Next())

			linkedLoop := linkedGP.First()
			if assert.NotNil(t, linkedLoop) {
				h3ind := h3s[0]

				bounds := ToGeoBoundary(h3ind)
				var linkedCoord *LinkedGeoCoord = linkedLoop.First()

				for linkedCoord != nil {
					var found bool
					for _, geo := range bounds {
						if geoCoordsAlmostEqual(geo, linkedCoord.Vertex) {
							found = true
							break
						}
					}
					assert.True(t, found)
					linkedCoord = linkedCoord.Next()
				}
			}
		}
	})
	t.Run("multiple", func(t *testing.T) {
		linkedGP := SetToLinkedGeo(validH3LinkedGeoPolygons)
		t.Run("First", func(t *testing.T) {
			first := linkedGP.First()
			assert.NotNil(t, first)

			t.Run("GeoLoop", func(t *testing.T) {
				t.Run("First", func(t *testing.T) {
					lFirst := first.First()
					assert.NotNil(t, lFirst)
				})
				t.Run("Last", func(t *testing.T) {
					lLast := first.Last()
					assert.NotNil(t, lLast)
				})
				t.Run("Next", func(t *testing.T) {
					lNext := first.Next()
					if assert.NotNil(t, lNext) {
						for lNext != nil {
							lNext = lNext.Next()
						}
					}
				})
			})
		})
		t.Run("Last", func(t *testing.T) {
			last := linkedGP.Last()
			assert.NotNil(t, last)
		})
		t.Run("Next", func(t *testing.T) {
			next := linkedGP.Next()
			assert.Nil(t, next)
		})
	})
}

func TestGeofence(t *testing.T) {
	t.Run("with geocoords", func(t *testing.T) {
		fence := &Geofence{
			GeoCoords: &GeoCoords{Coords: []*GeoCoord{{Latitude: 0.214, Longitude: 12.412}}},
		}
		cPtr := fence.toCPtr()
		gcPtr1 := fence.v.verts

		assert.NotNil(t, cPtr)
		defer fence.destroy()
		// call one more time should not create pointer again
		cPtr2 := fence.toCPtr()

		assert.Equal(t, cPtr, cPtr2)

		gcPtr := fence.GeoCoords.toCPtr()
		assert.Equal(t, gcPtr1, gcPtr)

	})

	t.Run("no geocoords", func(t *testing.T) {
		fence := &Geofence{}

		cPtr := fence.toCPtr()
		assert.NotNil(t, cPtr)
		defer fence.destroy()
		assert.Nil(t, fence.v.verts)
	})
}

func TestGeoCoords(t *testing.T) {
	gc := &GeoCoords{}

	gcCptr := gc.toCPtr()
	assert.Nil(t, gcCptr)
}

func TestLinkedGeoLoop(t *testing.T) {
	// Create new raw geoloop struct with no pointers
	gl := &LinkedGeoLoop{}
	assert.NotNil(t, gl.toCPtr())

	assert.Nil(t, gl.First())
	assert.Nil(t, gl.Last())
	assert.Nil(t, gl.Next())
}

func TestLinkedGeoPolygon(t *testing.T) {
	gp := &LinkedGeoPolygon{}
	assert.NotNil(t, gp.toCPtr())

	assert.Nil(t, gp.First())
	assert.Nil(t, gp.Last())
	assert.Nil(t, gp.Next())

	t.Run("next", func(t *testing.T) {
		nextLGP := &LinkedGeoPolygon{}
		gp.v.next = nextLGP.toCPtr()

		assert.NotNil(t, gp.Next())
	})

}

func TestPolyfill(t *testing.T) {
	t.Parallel()

	for i, fence := range validPolyfillFences {
		var res int
		switch i {
		case 0:
			res = 10
		case 1:
			res = 13
		case 2:
			res = 15
		}

		t.Run(fmt.Sprintf("polyfill %d", i), func(t *testing.T) {
			polygon := &GeoPolygon{Exterior: &(*fence)}
			if holes := validPolyfillHoles[i]; len(holes) > 0 {
				polygon.Holes = holes
			}

			h3Set := Polyfill(polygon, res)
			defer polygon.Destroy()

			assert.True(t, len(h3Set) <= MaxPolyfillSize(polygon, res))

			t.Logf("Polyfill %d, len: %d", i, len(h3Set))
		})
	}
}

func TestHexRing(t *testing.T) {
	t.Parallel()
	for k, expected := range validH3Rings1 {
		t.Run(fmt.Sprintf("ring size %d", k), func(t *testing.T) {
			actual, err := HexRing(validH3Index, k)
			require.NoError(t, err)
			assert.ElementsMatch(t, expected, actual)
		})
	}
	t.Run("pentagon err", func(t *testing.T) {
		t.Parallel()
		_, err := HexRing(pentagonH3Index, 1)
		assert.Error(t, err)
	})
}

func TestKRing(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		assertHexRange(t, validH3Rings1, KRing(validH3Index, len(validH3Rings1)-1))
	})
	t.Run("pentagon ok", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			KRing(pentagonH3Index, len(validH3Rings1)-1)
		})
	})
}

func TestKRingDistances(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		rings := KRingDistances(validH3Index, len(validH3Rings1)-1)
		for i, ring := range validH3Rings1 {
			assert.ElementsMatch(t, ring, rings[i])
		}
	})
	t.Run("pentagon ok", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			KRingDistances(pentagonH3Index, len(validH3Rings1)-1)
		})
	})
}

func TestHexRange(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		hexes, err := HexRange(validH3Index, len(validH3Rings1)-1)
		require.NoError(t, err)
		assertHexRange(t, validH3Rings1, hexes)
	})
	t.Run("pentagon err", func(t *testing.T) {
		t.Parallel()
		_, err := HexRange(pentagonH3Index, len(validH3Rings1)-1)
		assert.Error(t, err)
	})
}

func TestHexRangeDistances(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		rings, err := HexRangeDistances(validH3Index, len(validH3Rings1)-1)
		require.NoError(t, err)
		for i, ring := range validH3Rings1 {
			assert.ElementsMatch(t, ring, rings[i])
		}
	})
	t.Run("pentagon err", func(t *testing.T) {
		t.Parallel()
		_, err := HexRangeDistances(pentagonH3Index, len(validH3Rings1)-1)
		assert.Error(t, err)
	})
}

func TestHexRanges(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		hexranges, err := HexRanges(
			[]H3Index{
				validH3Rings1[0][0],
				validH3Rings2[0][0],
			}, len(validH3Rings2)-1)
		require.NoError(t, err)
		require.Len(t, hexranges, 2)
		assertHexRange(t, validH3Rings1, hexranges[0])
		assertHexRange(t, validH3Rings2, hexranges[1])
	})
	t.Run("pentagon err", func(t *testing.T) {
		_, err := HexRanges(
			[]H3Index{
				validH3Rings1[0][0],
				pentagonH3Index,
			}, len(validH3Rings2)-1)
		assert.Error(t, err)
		t.Parallel()
	})
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	assert.True(t, IsValid(validH3Index))
	assert.False(t, IsValid(0))
}

func TestFromGeoToGeo(t *testing.T) {
	t.Parallel()
	expectedGeo := GeoCoord{Latitude: 1, Longitude: 2}
	h := FromGeo(expectedGeo, 15)
	actualGeo := ToGeo(h)
	assertGeoCoord(t, expectedGeo, actualGeo)
}

func TestResolution(t *testing.T) {
	t.Parallel()
	for i := 1; i <= 15; i++ {
		h := FromGeo(validGeoCoord, i)
		assert.Equal(t, i, Resolution(h))
	}
}

func TestBaseCell(t *testing.T) {
	t.Parallel()
	bcID := BaseCell(validH3Index)
	assert.Equal(t, 6, bcID)
}

func TestToParent(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	parent := ToParent(validH3Index, Resolution(validH3Index)-1)

	// get the children at the resolution of the original index
	children := ToChildren(parent, Resolution(validH3Index))

	assertHexIn(t, validH3Index, children)
}

func TestCompact(t *testing.T) {
	t.Parallel()
	in := append([]H3Index{}, validH3Rings1[0][0])
	in = append(in, validH3Rings1[1]...)
	out := Compact(in)
	require.Len(t, out, 1)
	assert.Equal(t, ToParent(validH3Rings1[0][0], Resolution(validH3Rings1[0][0])-1), out[0])
}

func TestUncompact(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	res := Resolution(validH3Index) - 1
	parent := ToParent(validH3Index, res)

	out := Uncompact([]H3Index{parent}, res+1)
	assertHexIn(t, validH3Index, out)
}

func TestIsResClassIII(t *testing.T) {
	t.Parallel()
	res := Resolution(validH3Index) - 1
	parent := ToParent(validH3Index, res)

	assert.True(t, IsResClassIII(validH3Index))
	assert.False(t, IsResClassIII(parent))
}

func TestIsPentagon(t *testing.T) {
	t.Parallel()
	assert.False(t, IsPentagon(validH3Index))
	assert.True(t, IsPentagon(pentagonH3Index))
}

func TestAreNeighbors(t *testing.T) {
	t.Parallel()
	assert.False(t, AreNeighbors(pentagonH3Index, validH3Index))
	assert.True(t, AreNeighbors(validH3Rings1[1][0], validH3Rings1[1][1]))
}

func TestUnidirectionalEdge(t *testing.T) {
	t.Parallel()
	origin := validH3Rings1[1][0]
	destination := validH3Rings1[1][1]
	edge := UnidirectionalEdge(origin, destination)

	t.Run("is valid", func(t *testing.T) {
		t.Parallel()
		assert.True(t, UnidirectionalEdgeIsValid(edge))
		assert.False(t, UnidirectionalEdgeIsValid(validH3Index))
	})
	t.Run("get origin/destination from edge", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, origin, OriginFromUnidirectionalEdge(edge))
		assert.Equal(t, destination, DestinationFromUnidirectionalEdge(edge))

		// shadow origin/destination
		origin, destination := FromUnidirectionalEdge(edge)
		assert.Equal(t, origin, OriginFromUnidirectionalEdge(edge))
		assert.Equal(t, destination, DestinationFromUnidirectionalEdge(edge))
	})
	t.Run("get edges from hexagon", func(t *testing.T) {
		t.Parallel()
		edges := ToUnidirectionalEdges(validH3Index)
		assert.Len(t, edges, 6, "hexagon has 6 edges")
	})
	t.Run("get edges from pentagon", func(t *testing.T) {
		t.Parallel()
		edges := ToUnidirectionalEdges(pentagonH3Index)
		require.Len(t, edges, 5, "pentagon has 5 edges")
	})
	t.Run("get boundary from edge", func(t *testing.T) {
		t.Parallel()
		gb := UnidirectionalEdgeBoundary(edge)
		assert.Len(t, gb, 2)
	})
}

func TestString(t *testing.T) {
	t.Parallel()
	t.Run("bad string", func(t *testing.T) {
		t.Parallel()
		h := FromString("oops")
		assert.Equal(t, H3Index(0), h)
	})
	t.Run("good string round trip", func(t *testing.T) {
		t.Parallel()
		h := FromString(ToString(validH3Index))
		assert.Equal(t, validH3Index, h)
	})
	t.Run("no 0x prefix", func(t *testing.T) {
		t.Parallel()
		h3addr := ToString(validH3Index)
		assert.Equal(t, "850dab63fffffff", h3addr)
	})
}

func almostEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.InEpsilon(t, expected, actual, eps, msgAndArgs...)
}

func assertGeoCoord(t *testing.T, expected, actual GeoCoord) {
	almostEqual(t, expected.Latitude, actual.Latitude, "latitude mismatch")
	almostEqual(t, expected.Longitude, actual.Longitude, "longitude mismatch")
}

func assertGeoCoords(t *testing.T, expected, actual []GeoCoord) {
	for i, gc := range expected {
		assertGeoCoord(t, gc, actual[i])
	}
}

func assertHexRange(t *testing.T, expected [][]H3Index, actual []H3Index) {
	for i, ring := range expected {
		// each ring should be sorted by value because the order of a ring is
		// undefined.
		lower := rangeSize(i) - ringSize(i)
		upper := rangeSize(i)
		assert.ElementsMatch(t, ring, actual[lower:upper])
	}
}

func assertHexIn(t *testing.T, needle H3Index, haystack []H3Index) {
	var found bool
	for _, h := range haystack {
		found = needle == h
		if found {
			break
		}
	}
	assert.True(t, found,
		"expected %+v in %+v",
		needle, haystack)
}

func validHexRange() []H3Index {
	out := []H3Index{}
	for _, ring := range validH3Rings1 {
		out = append(out, ring...)
	}
	return out
}

func sortHexes(s []H3Index) []H3Index {
	sort.SliceStable(s, func(i, j int) bool {
		return s[i] < s[j]
	})
	return s
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func floatsAlmostEqual(first, second float64) bool {
	actualEps := math.Abs(first-second) / math.Abs(first)
	return actualEps <= eps
}

func geoCoordsAlmostEqual(first, second GeoCoord) bool {
	latOk := floatsAlmostEqual(first.Latitude, second.Latitude)
	lonOk := floatsAlmostEqual(first.Longitude, second.Longitude)
	return latOk && lonOk
}
