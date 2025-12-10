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
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"testing"
)

const eps = 1e-4

// validH3Index resolution 5.
const (
	validCell     = Cell(0x850dab63fffffff)
	pentagonCell  = Cell(0x821c07fffffffff)
	lineStartCell = Cell(0x89283082803ffff)
	lineEndCell   = Cell(0x8929a5653c3ffff)
	validVertex   = Vertex(0x2050dab63fffffff)
)

var (
	validDiskDist3_1 = [][]Cell{
		{
			validCell,
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

	validLatLng1 = LatLng{
		Lat: 67.1509268640,
		Lng: -168.3908885810,
	}
	validLatLng2 = LatLng{
		Lat: 37.775705522929044,
		Lng: -122.41812765598296,
	}

	// validGeoLoop is the boundary of validCell_1.
	validGeoLoop = GeoLoop{
		{Lat: 67.224749856, Lng: -168.523006585},
		{Lat: 67.140938355, Lng: -168.626914333},
		{Lat: 67.067252558, Lng: -168.494913285},
		{Lat: 67.077062918, Lng: -168.259695931},
		{Lat: 67.160561948, Lng: -168.154801171},
		{Lat: 67.234563187, Lng: -168.286102782},
	}

	validHole1 = GeoLoop{
		{Lat: 67.2, Lng: -168.4},
		{Lat: 67.1, Lng: -168.4},
		{Lat: 67.1, Lng: -168.3},
		{Lat: 67.2, Lng: -168.3},
	}

	validHole2 = GeoLoop{
		{Lat: 67.21, Lng: -168.41},
		{Lat: 67.22, Lng: -168.41},
		{Lat: 67.22, Lng: -168.42},
	}

	validGeoPolygonNoHoles = GeoPolygon{GeoLoop: validGeoLoop}

	validGeoPolygonHoles = GeoPolygon{
		GeoLoop: validGeoLoop,
		Holes: []GeoLoop{
			validHole1,
			validHole2,
		},
	}

	validEdge = DirectedEdge(0x1250dab73fffffff)
)

func TestLatLngToCell(t *testing.T) {
	t.Parallel()

	c, err := LatLngToCell(validLatLng1, 5)
	assertEqual(t, validCell, c)
	assertNoErr(t, err)

	_, err = LatLngToCell(NewLatLng(0, 0), MaxResolution+1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestCellToLatLng(t *testing.T) {
	t.Parallel()

	g, err := CellToLatLng(validCell)
	assertEqualLatLng(t, validLatLng1, g)
	assertNoErr(t, err)

	_, err = CellToLatLng(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestToCellBoundary(t *testing.T) {
	t.Parallel()

	boundary, err := validCell.Boundary()
	assertEqualLatLngs(t, validGeoLoop[:], boundary[:])
	assertNoErr(t, err)

	c := Cell(-1)
	_, err = c.Boundary()
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCellToLocalIJ(t *testing.T) {
	t.Parallel()

	_, err := CellToLocalIJ(validCell, validCell)
	assertNoErr(t, err)

	_, err = CellToLocalIJ(-1, -1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestLocalIJToCell(t *testing.T) {
	t.Parallel()

	ij, _ := CellToLocalIJ(validCell, validCell)
	c, err := LocalIJToCell(validCell, ij)
	assertNoErr(t, err)
	assertEqual(t, c, validCell)

	_, err = LocalIJToCell(-1, ij)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestGridDisk(t *testing.T) {
	t.Parallel()

	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()

		gd, err := validCell.GridDisk(len(validDiskDist3_1) - 1)
		assertEqualDisks(t,
			flattenDisks(validDiskDist3_1),
			gd,
		)
		assertNoErr(t, err)
	})

	t.Run("pentagon ok", func(t *testing.T) {
		t.Parallel()

		assertNoPanic(t, func() {
			disk, err := GridDisk(pentagonCell, 1)
			assertEqual(t, 6, len(disk), "expected pentagon disk to have 6 cells")
			assertNoErr(t, err)
		})
	})

	t.Run("invalid cell", func(t *testing.T) {
		t.Parallel()

		c := Cell(-1)
		_, err := c.GridDisk(1)
		assertErr(t, err)
		assertErrIs(t, err, ErrCellInvalid)
	})
}

func TestGridDisksUnsafe(t *testing.T) {
	t.Parallel()

	t.Run("two cells", func(t *testing.T) {
		t.Parallel()

		gds, err := GridDisksUnsafe([]Cell{validCell, validCell}, len(validDiskDist3_1)-1)
		assertNoErr(t, err)
		assertEqual(t, 2, len(gds), "expected grid disks to have two arrays returned")
		assertEqualDisks(t,
			flattenDisks(validDiskDist3_1),
			gds[0],
			"expected grid disks[0] to be the same",
		)
		assertEqualDisks(t,
			flattenDisks(validDiskDist3_1),
			gds[1],
			"expected grid disks[1] to be the same",
		)
	})

	t.Run("pentagon", func(t *testing.T) {
		t.Parallel()

		_, err := GridDisksUnsafe([]Cell{pentagonCell}, 1)
		assertErr(t, err)
		assertErrIs(t, err, ErrPentagon)
	})

	t.Run("invalid cell", func(t *testing.T) {
		t.Parallel()

		c := Cell(-1)
		_, err := GridDisksUnsafe([]Cell{c}, 1)
		assertErr(t, err)
		assertErrIs(t, err, ErrCellInvalid)
	})

	t.Run("invalid k", func(t *testing.T) {
		t.Parallel()

		_, err := GridDisksUnsafe([]Cell{validCell}, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
	})
}

func TestGridDiskDistances(t *testing.T) {
	t.Parallel()

	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		rings, err := validCell.GridDiskDistances(len(validDiskDist3_1) - 1)
		assertNoErr(t, err)
		assertEqualDiskDistances(t, validDiskDist3_1, rings)

		rings, err = validCell.GridDiskDistancesSafe(len(validDiskDist3_1) - 1)
		assertNoErr(t, err)
		assertEqualDiskDistances(t, validDiskDist3_1, rings)
	})

	t.Run("pentagon centered", func(t *testing.T) {
		t.Parallel()
		assertNoPanic(t, func() {
			rings, err := GridDiskDistances(pentagonCell, 1)
			assertNoErr(t, err)
			assertEqual(t, 2, len(rings), "expected 2 rings")
			assertEqual(t, 5, len(rings[1]), "expected 5 cells in second ring")

			rings, err = GridDiskDistancesSafe(pentagonCell, 1)
			assertNoErr(t, err)
			assertEqual(t, 2, len(rings), "expected 2 rings")
			assertEqual(t, 5, len(rings[1]), "expected 5 cells in second ring")
		})
	})

	t.Run("invalid k-ring", func(t *testing.T) {
		rings, err := GridDiskDistances(pentagonCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
		assertNil(t, rings)

		rings, err = GridDiskDistancesSafe(pentagonCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
		assertNil(t, rings)
	})
}

func TestGridDiskDistancesUnsafe(t *testing.T) {
	t.Parallel()

	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		rings, err := validCell.GridDiskDistancesUnsafe(len(validDiskDist3_1) - 1)
		assertNoErr(t, err)
		assertEqualDiskDistances(t, validDiskDist3_1, rings)
	})

	t.Run("pentagon centered", func(t *testing.T) {
		t.Parallel()
		assertNoPanic(t, func() {
			_, err := GridDiskDistancesUnsafe(pentagonCell, 1)
			assertErr(t, err)
			assertErrIs(t, err, ErrPentagon)
		})
	})

	t.Run("invalid k-ring", func(t *testing.T) {
		rings, err := GridDiskDistancesUnsafe(pentagonCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
		assertNil(t, rings)
	})
}

func TestGridRing(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		gr, err := validCell.GridRing(1)
		assertEqualDisks(t,
			validDiskDist3_1[1],
			gr,
		)
		assertNoErr(t, err)
	})

	t.Run("success/pentagon", func(t *testing.T) {
		t.Parallel()

		gr, err := GridRing(pentagonCell, 1)
		assertEqual(t, 5, len(gr))
		assertNoErr(t, err)
	})

	t.Run("err/invalid_cell", func(t *testing.T) {
		t.Parallel()

		c := Cell(-1)
		_, err := c.GridRing(1)
		assertErr(t, err)
		assertErrIs(t, err, ErrCellInvalid)
	})

	t.Run("err/invalid_kring", func(t *testing.T) {
		rings, err := GridRing(pentagonCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
		assertNil(t, rings)
	})
}

func TestGridRingUnsafe(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		gr, err := validCell.GridRingUnsafe(1)
		assertEqualDisks(t,
			validDiskDist3_1[1],
			gr,
		)
		assertNoErr(t, err)
	})

	t.Run("err/invalid_k", func(t *testing.T) {
		_, err := GridRingUnsafe(validCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
	})

	t.Run("err/pentagon", func(t *testing.T) {
		t.Parallel()

		_, err := GridRingUnsafe(pentagonCell, 1)
		assertErr(t, err)
		assertErrIs(t, err, ErrPentagon)
	})

	t.Run("err/invalid_cell", func(t *testing.T) {
		t.Parallel()

		c := Cell(-1)
		_, err := c.GridRingUnsafe(1)
		assertErr(t, err)
		assertErrIs(t, err, ErrCellInvalid)
	})
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	assertTrue(t, validCell.IsValid())
	assertFalse(t, Cell(0).IsValid())
}

func TestRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("latlng", func(t *testing.T) {
		t.Parallel()
		expectedGeo := LatLng{Lat: 1, Lng: 2}
		c, _ := LatLngToCell(expectedGeo, MaxResolution)
		actualGeo, _ := CellToLatLng(c)
		assertEqualLatLng(t, expectedGeo, actualGeo)

		expectedCell, _ := expectedGeo.Cell(MaxResolution)
		expectedLatLng, _ := expectedCell.LatLng()
		assertEqualLatLng(t, expectedGeo, expectedLatLng)
	})

	t.Run("cell", func(t *testing.T) {
		t.Parallel()
		geo, _ := CellToLatLng(validCell)
		actualCell, _ := LatLngToCell(geo, validCell.Resolution())
		assertEqual(t, validCell, actualCell)
	})
}

func TestResolution(t *testing.T) {
	t.Parallel()

	for i := 1; i <= MaxResolution; i++ {
		c, _ := LatLngToCell(validLatLng1, i)
		assertEqual(t, i, c.Resolution())
	}

	edges, _ := validCell.DirectedEdges()
	for _, e := range edges {
		assertEqual(t, validCell.Resolution(), e.Resolution())
	}

	vertexes, _ := validCell.Vertexes()
	for _, v := range vertexes {
		assertEqual(t, validCell.Resolution(), v.Resolution())
	}
}

func TestBaseCellNumber(t *testing.T) {
	t.Parallel()
	bcID := validCell.BaseCellNumber()
	assertEqual(t, 6, bcID)
}

func TestParent(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	parent, err := validCell.ImmediateParent()
	assertNoErr(t, err)

	// get the children at the resolution of the original index
	children, _ := parent.ImmediateChildren()

	assertCellIn(t, validCell, children)

	_, err = validCell.Parent(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestChildren_Error(t *testing.T) {
	t.Parallel()

	children, err := validCell.Children(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
	assertNil(t, children)
}

func TestCompactCells(t *testing.T) {
	t.Parallel()

	in := flattenDisks(validDiskDist3_1[:2])
	t.Logf("in: %v", in)
	out, err := CompactCells(in)
	t.Logf("out: %v", in)
	assertNoErr(t, err)
	assertEqual(t, 1, len(out))

	p, _ := validDiskDist3_1[0][0].ImmediateParent()
	assertEqual(t, p, out[0])

	_, err = CompactCells([]Cell{-1})
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestUncompactCells(t *testing.T) {
	t.Parallel()

	// get the index's parent by requesting that index's resolution+1
	parent, _ := validCell.ImmediateParent()
	out, err := UncompactCells([]Cell{parent}, parent.Resolution()+1)
	assertNoErr(t, err)
	assertCellIn(t, validCell, out)

	out, err = UncompactCells([]Cell{parent}, -1)
	assertErr(t, err)
	assertErrIs(t, err, ErrRsolutionMismatch)
	assertNil(t, out)
}

func TestChildPosToCell(t *testing.T) {
	t.Parallel()

	children, _ := validCell.Children(6)

	cell, err := validCell.ChildPosToCell(0, 6)
	assertNoErr(t, err)
	assertEqual(t, children[0], cell)

	cell, err = ChildPosToCell(0, validCell, 6)
	assertNoErr(t, err)
	assertEqual(t, children[0], cell)

	_, err = validCell.ChildPosToCell(0, -1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestChildPos(t *testing.T) {
	t.Parallel()

	children, _ := validCell.Children(7)

	pos, err := children[32].ChildPos(validCell.Resolution())
	assertNoErr(t, err)
	assertEqual(t, 32, pos)

	pos, err = CellToChildPos(children[32], validCell.Resolution())
	assertNoErr(t, err)
	assertEqual(t, 32, pos)

	_, err = validCell.ChildPos(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestIsResClassIII(t *testing.T) {
	t.Parallel()

	p, _ := validCell.ImmediateParent()
	assertTrue(t, validCell.IsResClassIII())
	assertFalse(t, p.IsResClassIII())
}

func TestIsPentagon(t *testing.T) {
	t.Parallel()
	assertFalse(t, validCell.IsPentagon())
	assertTrue(t, pentagonCell.IsPentagon())
}

func TestIsNeighbor(t *testing.T) {
	t.Parallel()

	isNeighbor, err := validCell.IsNeighbor(pentagonCell)
	assertErr(t, err)
	assertErrIs(t, err, ErrRsolutionMismatch)
	assertFalse(t, isNeighbor)

	edges, _ := validCell.DirectedEdges()
	dest, _ := edges[0].Destination()
	isNeighbor, err = dest.IsNeighbor(validCell)
	assertNoErr(t, err)
	assertTrue(t, isNeighbor)
}

func TestDirectedEdge(t *testing.T) {
	t.Parallel()

	origin := validDiskDist3_1[1][0]
	edges, err := origin.DirectedEdges()
	assertNoErr(t, err)

	destination, err := edges[0].Destination()
	assertNoErr(t, err)

	edge, err := origin.DirectedEdge(destination)
	assertNoErr(t, err)

	t.Run("is valid", func(t *testing.T) {
		t.Parallel()
		assertTrue(t, edge.IsValid())
		assertFalse(t, DirectedEdge(validCell).IsValid())
	})

	t.Run("get origin/destination from edge", func(t *testing.T) {
		t.Parallel()
		edgeOrigin, err := edge.Origin()
		assertNoErr(t, err)
		assertEqual(t, origin, edgeOrigin)

		edgeDestination, err := edge.Destination()
		assertNoErr(t, err)
		assertEqual(t, destination, edgeDestination)

		// shadow origin/destination
		cells, err := edge.Cells()
		assertNoErr(t, err)

		origin, destination := cells[0], cells[1]
		assertEqual(t, origin, edgeOrigin)
		assertEqual(t, destination, edgeDestination)
	})

	t.Run("edge cells error", func(t *testing.T) {
		t.Parallel()
		cells, err := DirectedEdge(-1).Cells()
		assertErr(t, err)
		assertErrIs(t, err, ErrDirectedEdgeInvalid)
		assertNil(t, cells)
	})

	t.Run("get edges from hexagon", func(t *testing.T) {
		t.Parallel()
		edges, err := validCell.DirectedEdges()
		assertNoErr(t, err)
		assertEqual(t, 6, len(edges), "hexagon has 6 edges")
	})

	t.Run("get edges from pentagon", func(t *testing.T) {
		t.Parallel()
		edges, err := pentagonCell.DirectedEdges()
		assertNoErr(t, err)
		assertEqual(t, 5, len(edges), "pentagon has 5 edges")
	})

	t.Run("get boundary from edge", func(t *testing.T) {
		t.Parallel()
		gb, err := edge.Boundary()
		assertNoErr(t, err)
		assertEqual(t, 2, len(gb), "edge has 2 boundary cells")
	})

	t.Run("boundary error", func(t *testing.T) {
		t.Parallel()
		gb, err := DirectedEdge(-1).Boundary()
		assertErr(t, err)
		assertErrIs(t, err, ErrDirectedEdgeInvalid)
		assertNil(t, gb)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		_, err := validCell.DirectedEdge(-1)
		assertErr(t, err)
		assertErrIs(t, err, ErrNotNeighbors)
	})
}

func TestStrings(t *testing.T) {
	t.Parallel()

	t.Run("bad string", func(t *testing.T) {
		t.Parallel()
		i := IndexFromString("oops")
		assertEqual(t, 0, i)

		c := CellFromString("oops")
		assertEqual(t, 0, c)
	})

	t.Run("good string round trip", func(t *testing.T) {
		t.Parallel()
		i := IndexFromString(validCell.String())

		assertEqual(t, validCell, Cell(i))

		c := CellFromString(validCell.String())
		assertEqual(t, validCell, c)
	})

	t.Run("no 0x prefix", func(t *testing.T) {
		t.Parallel()
		h3addr := validCell.String()
		assertEqual(t, "850dab63fffffff", h3addr)
	})

	t.Run("marshalling text", func(t *testing.T) {
		t.Parallel()
		c := Cell(0)
		text, err := validCell.MarshalText()
		assertNoErr(t, err)

		err = c.UnmarshalText([]byte("0x" + string(text)))
		assertNoErr(t, err)
		assertEqual(t, validCell, c)

		err = c.UnmarshalText([]byte(""))
		assertErr(t, err)
	})
}

func TestPolygonToCells(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		cells, err := PolygonToCells(GeoPolygon{}, 6)
		assertNoErr(t, err)
		assertEqual(t, 0, len(cells))
	})

	t.Run("without holes", func(t *testing.T) {
		t.Parallel()

		cells, err := validGeoPolygonNoHoles.Cells(6)
		assertNoErr(t, err)

		expectedIndexes := []Cell{
			0x860dab607ffffff,
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertEqualCells(t, expectedIndexes, cells)
	})

	t.Run("with hole", func(t *testing.T) {
		t.Parallel()

		cells, err := validGeoPolygonHoles.Cells(6)
		assertNoErr(t, err)

		expectedIndexes := []Cell{
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertEqualCells(t, expectedIndexes, cells)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		cells, err := validGeoPolygonHoles.Cells(-1)
		assertErr(t, err)
		assertErrIs(t, err, ErrResolutionDomain)
		assertNil(t, cells)
	})
}

func TestCellsToMultiPolygon(t *testing.T) {
	t.Parallel()

	// 7 cells in disk -> 1 polygon, 18-point loop, and no holes
	c, _ := LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ := GridDisk(c, 1)
	res, err := CellsToMultiPolygon(cells)
	assertNoErr(t, err)
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 0)

	// 6 cells in ring -> 1 polygon, 18-point loop, and 1 6-point hole
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ = GridDisk(c, 1)
	res, err = CellsToMultiPolygon(cells[1:])
	assertNoErr(t, err)
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 1)
	assertEqual(t, len(res[0].Holes[0]), 6)

	// 2 hexagons connected -> 1 polygon, 10-point loop (2 shared points) and no holes
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ = GridDisk(c, 1)
	res, err = CellsToMultiPolygon(cells[:2])
	assertNoErr(t, err)
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 10)
	assertEqual(t, len(res[0].Holes), 0)

	// 2 distinct disks -> 2 polygons, 2 18-point loops, and no holes
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells1, _ := GridDisk(c, 1)

	c, _ = LatLngToCell(NewLatLng(10, 10), 10)
	cells2, _ := GridDisk(c, 1)
	cells = append(cells1, cells2...)
	res, err = CellsToMultiPolygon(cells)
	assertNoErr(t, err)
	assertEqual(t, len(res), 2)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 0)
	assertEqual(t, len(res[1].GeoLoop), 18)
	assertEqual(t, len(res[1].Holes), 0)

	// empty
	res, err = CellsToMultiPolygon([]Cell{})
	assertNoErr(t, err)
	assertEqual(t, len(res), 0)

	// Error.
	res, err = CellsToMultiPolygon([]Cell{-1})
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
	assertNil(t, res)
}

func TestPolygonToCellsExperimental(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		for _, flag := range []ContainmentMode{
			ContainmentCenter,
			ContainmentFull,
			ContainmentOverlapping,
			ContainmentOverlappingBbox,
		} {
			cells, err := PolygonToCellsExperimental(GeoPolygon{}, 6, flag)
			if err != nil {
				t.Error(t)
			}

			assertEqual(t, 0, len(cells))
		}
	})

	t.Run("without holes", func(t *testing.T) {
		t.Parallel()

		for _, flag := range []ContainmentMode{
			ContainmentCenter,
			ContainmentFull,
			ContainmentOverlapping,
			ContainmentOverlappingBbox,
		} {
			cells, err := PolygonToCellsExperimental(validGeoPolygonNoHoles, 6, flag)
			if err != nil {
				t.Error(t)
			}
			expectedCellCounts := map[ContainmentMode]int{
				ContainmentCenter:          7,
				ContainmentFull:            1,
				ContainmentOverlapping:     14,
				ContainmentOverlappingBbox: 21,
			}
			assertEqual(t, expectedCellCounts[flag], len(cells))
		}
	})

	t.Run("with holes", func(t *testing.T) {
		t.Parallel()

		for _, flag := range []ContainmentMode{
			ContainmentCenter,
			ContainmentFull,
			ContainmentOverlapping,
			ContainmentOverlappingBbox,
		} {
			cells, err := PolygonToCellsExperimental(validGeoPolygonHoles, 6, flag)
			if err != nil {
				t.Error(t)
			}
			expectedCellCounts := map[ContainmentMode]int{
				ContainmentCenter:          6,
				ContainmentFull:            0,
				ContainmentOverlapping:     14,
				ContainmentOverlappingBbox: 21,
			}

			assertEqual(t, expectedCellCounts[flag], len(cells))
		}
	})

	t.Run("busting memory", func(t *testing.T) {
		t.Parallel()

		for _, flag := range []ContainmentMode{
			ContainmentCenter,
			ContainmentOverlapping,
			ContainmentOverlappingBbox,
		} {
			_, err := PolygonToCellsExperimental(validGeoPolygonHoles, 6, flag, 3)
			if err != ErrMemoryBounds {
				t.Error(t)
			}
		}
	})

	t.Run("err/invalid_containment_mode", func(t *testing.T) {
		t.Parallel()

		_, err := PolygonToCellsExperimental(validGeoPolygonHoles, 6, ContainmentInvalid)
		assertErr(t, err)
		assertErrIs(t, err, ErrOptionInvalid)
	})
}

func TestGridPath(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		path, err := lineStartCell.GridPath(lineEndCell)

		assertNoErr(t, err)
		assertEqual(t, lineStartCell, path[0])
		assertEqual(t, lineEndCell, path[len(path)-1])

		for i := 0; i < len(path)-1; i++ {
			isNeighbor, _ := path[i].IsNeighbor(path[i+1])
			assertTrue(t, isNeighbor)
		}
	})

	t.Run("err/res_mismatch", func(t *testing.T) {
		t.Parallel()

		_, err := GridPath(1, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrRsolutionMismatch)
	})

	t.Run("err/failed", func(t *testing.T) {
		t.Parallel()

		c1, _ := NewLatLng(1, 1).Cell(5)
		c2, _ := NewLatLng(50.10320148224132, -143.47849001502516).Cell(5)
		_, err := GridPath(c1, c2)
		assertErr(t, err)
		assertErrIs(t, err, ErrFailed)
	})

	t.Run("err/pentagon", func(t *testing.T) {
		t.Parallel()

		start := Cell(IndexFromString("0x820807fffffffff"))
		end := Cell(IndexFromString("0x8208e7fffffffff"))

		_, err := GridPath(start, end)
		assertErr(t, err)
		assertErrIs(t, err, ErrPentagon)
	})
}

func TestHexAreaKm2(t *testing.T) {
	t.Parallel()

	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(0)
		assertNoErr(t, err)
		assertEqualEps(t, float64(4357449.4161), area)
	})

	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(15)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.0000009), area)
	})

	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(8)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.7373276), area)
	})

	t.Run("resolution error", func(t *testing.T) {
		t.Parallel()
		_, err := HexagonAreaAvgKm2(-1)
		assertErr(t, err)
		assertErrIs(t, err, ErrResolutionDomain)
	})
}

func TestHexAreaM2(t *testing.T) {
	t.Parallel()

	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(0)
		assertNoErr(t, err)
		assertEqualEps(t, float64(4357449416078.3901), area)
	})

	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(15)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.8953), area)
	})

	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(8)
		assertNoErr(t, err)
		assertEqualEps(t, float64(737327.5976), area)
	})

	t.Run("resolution error", func(t *testing.T) {
		t.Parallel()
		_, err := HexagonAreaAvgM2(-1)
		assertErr(t, err)
		assertErrIs(t, err, ErrResolutionDomain)
	})
}

func TestPointDistRads(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceRads(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(0.6796147656451452), distance)
}

func TestPointDistKm(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceKm(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(4329.830552183446), distance)
}

func TestPointDistM(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceM(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(4329830.5521834465), distance)
}

func TestCellAreaRads2(t *testing.T) {
	t.Parallel()
	area, err := CellAreaRads2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(0.000006643967854567278), area)

	_, err = CellAreaRads2(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCellAreaKm2(t *testing.T) {
	t.Parallel()
	area, err := CellAreaKm2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(269.6768779509321), area)

	_, err = CellAreaKm2(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCellAreaM2(t *testing.T) {
	t.Parallel()
	area, err := CellAreaM2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(269676877.95093215), area)

	_, err = CellAreaM2(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestHexagonEdgeLengthKm(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()

		length, err := HexagonEdgeLengthAvgKm(0)
		assertNoErr(t, err)
		assertEqual(t, 1281.256011, length)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()

		length, err := HexagonEdgeLengthAvgKm(15)
		assertNoErr(t, err)
		assertEqual(t, 0.000584169, length)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()

		length, err := HexagonEdgeLengthAvgKm(8)
		assertNoErr(t, err)
		assertEqual(t, 0.53141401, length)
	})
}

func TestHexagonEdgeLengthM(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(0)
		assertNoErr(t, err)
		assertEqual(t, 1281256.011, area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(15)
		assertNoErr(t, err)
		assertEqual(t, 0.584168630, area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(8)
		assertNoErr(t, err)
		assertEqual(t, 531.4140101, area)
	})
	t.Run("invalid resolution", func(t *testing.T) {
		t.Parallel()
		_, err := HexagonEdgeLengthAvgM(-1)
		assertErr(t, err)
		assertErrIs(t, err, ErrResolutionDomain)
	})
}

func TestEdgeLengthRads(t *testing.T) {
	t.Parallel()
	length, err := EdgeLengthRads(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(0.001569665746947077), length)

	_, err = EdgeLengthRads(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrDirectedEdgeInvalid)
}

func TestEdgeLengthKm(t *testing.T) {
	t.Parallel()

	distance, err := EdgeLengthKm(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(10.00035174544159), distance)

	_, err = EdgeLengthKm(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrDirectedEdgeInvalid)
}

func TestEdgeLengthM(t *testing.T) {
	t.Parallel()

	distance, err := EdgeLengthM(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(10000.351745441589), distance)

	_, err = EdgeLengthM(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrDirectedEdgeInvalid)
}

func TestNumCells(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 122, NumCells(0))
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 569707381193162, NumCells(15))
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 691776122, NumCells(8))
	})
}

func TestRes0Cells(t *testing.T) {
	t.Parallel()
	cells, err := Res0Cells()

	assertNoErr(t, err)
	assertEqual(t, 122, len(cells))
	assertEqual(t, Cell(0x8001fffffffffff), cells[0])
	assertEqual(t, Cell(0x80f3fffffffffff), cells[121])
}

func TestGridDistance(t *testing.T) {
	t.Parallel()

	dist, err := lineStartCell.GridDistance(lineEndCell)
	assertEqual(t, 1823, dist)
	assertNoErr(t, err)

	_, err = GridDistance(-1, -2)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCenterChild(t *testing.T) {
	t.Parallel()

	child, err := validCell.CenterChild(15)
	assertNoErr(t, err)
	assertEqual(t, Cell(0x8f0dab600000000), child)

	_, err = validCell.CenterChild(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestIcosahedronFaces(t *testing.T) {
	t.Parallel()

	faces, err := validDiskDist3_1[1][1].IcosahedronFaces()

	assertEqual(t, 1, len(faces))
	assertEqual(t, 1, faces[0])
	assertNoErr(t, err)

	c := Cell(-1)

	_, err = c.IcosahedronFaces()
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestPentagons(t *testing.T) {
	t.Parallel()

	for _, res := range []int{0, 8, 15} {
		t.Run(fmt.Sprintf("res=%d", res), func(t *testing.T) {
			t.Parallel()

			pentagons, err := Pentagons(res)
			assertNoErr(t, err)
			assertEqual(t, 12, len(pentagons))

			for _, pentagon := range pentagons {
				assertTrue(t, pentagon.IsPentagon())
				assertEqual(t, res, pentagon.Resolution())
			}
		})
	}

	_, err := Pentagons(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestCellToVertex(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		cell           Cell
		vertexNum      int
		expectedVertex Vertex
		expectedErr    error
	}{
		"success":             {cell: validCell, vertexNum: 0, expectedVertex: validVertex, expectedErr: nil},
		"err/cell_domain":     {cell: validCell, vertexNum: 6, expectedVertex: 0, expectedErr: ErrDomain},    // vertex num should be between 0 and 5 for hexagonal cells.
		"err/pentagon_domain": {cell: pentagonCell, vertexNum: 5, expectedVertex: 0, expectedErr: ErrDomain}, // vertex num should be between 0 and 4 for pentagon cells.
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			vertex, err := tc.cell.Vertex(tc.vertexNum)
			assertErrIs(t, err, tc.expectedErr)
			assertEqual(t, tc.expectedVertex, vertex)
		})
	}
}

func TestCellToVertexes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		cell        Cell
		numVertexes int
		expectedErr error
	}{
		"cell":     {cell: validCell, numVertexes: 6, expectedErr: nil},
		"pentagon": {cell: pentagonCell, numVertexes: 5, expectedErr: nil},
		"invalid":  {cell: -1, numVertexes: 0, expectedErr: ErrFailed}, // Invalid cell.
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			vertexes, err := tc.cell.Vertexes()
			assertErrIs(t, err, tc.expectedErr)
			assertEqual(t, tc.numVertexes, len(vertexes))
		})
	}
}

func TestVertexToLatLng(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		vertex         Vertex
		expectedLatLng LatLng
		expectedErr    error
	}{
		"success":     {vertex: validVertex, expectedLatLng: LatLng{Lat: 67.22475, Lng: -168.52301}, expectedErr: nil},
		"err/invalid": {vertex: -1, expectedLatLng: LatLng{}, expectedErr: ErrCellInvalid}, // Invalid vertex.
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			latLng, err := tc.vertex.LatLng()
			assertErrIs(t, err, tc.expectedErr)
			assertEqualLatLng(t, tc.expectedLatLng, latLng)
		})
	}
}

func TestIsValidVertex(t *testing.T) {
	t.Parallel()

	assertFalse(t, IsValidVertex(0))
	assertTrue(t, IsValidVertex(2473183460575936511))
	assertTrue(t, validVertex.IsValid())
}

func TestVertex_Strings(t *testing.T) {
	t.Parallel()

	t.Run("bad string", func(t *testing.T) {
		t.Parallel()
		v := VertexFromString("invalid")
		assertEqual(t, 0, v)
	})

	t.Run("good string round trip", func(t *testing.T) {
		t.Parallel()
		v := VertexFromString(validVertex.String())
		assertEqual(t, validVertex, v)
	})

	t.Run("no 0x prefix", func(t *testing.T) {
		t.Parallel()
		h3addr := validVertex.String()
		assertEqual(t, "2050dab63fffffff", h3addr)
	})

	t.Run("marshalling text", func(t *testing.T) {
		t.Parallel()
		text, err := validVertex.MarshalText()
		assertNoErr(t, err)

		var v Vertex
		err = v.UnmarshalText([]byte("0x" + string(text)))
		assertNoErr(t, err)
		assertEqual(t, validVertex, v)

		err = v.UnmarshalText([]byte(""))
		assertErr(t, err)
	})
}

func TestDirectedEdge_Strings(t *testing.T) {
	t.Parallel()

	t.Run("bad string", func(t *testing.T) {
		t.Parallel()
		e := DirectedEdgeFromString("invalid")
		assertEqual(t, 0, e)
	})

	t.Run("good string round trip", func(t *testing.T) {
		t.Parallel()
		e := DirectedEdgeFromString(validEdge.String())
		assertEqual(t, validEdge, e)
	})

	t.Run("no 0x prefix", func(t *testing.T) {
		t.Parallel()
		h3addr := validEdge.String()
		assertEqual(t, "1250dab73fffffff", h3addr)
	})

	t.Run("marshalling text", func(t *testing.T) {
		t.Parallel()
		text, err := validEdge.MarshalText()
		assertNoErr(t, err)

		var e DirectedEdge
		err = e.UnmarshalText([]byte("0x" + string(text)))
		assertNoErr(t, err)
		assertEqual(t, validEdge, e)

		err = e.UnmarshalText([]byte(""))
		assertErr(t, err)
	})
}

func TestStrings_Deprecated(t *testing.T) {
	s := CellToString(validCell)
	assertEqual(t, "850dab63fffffff", s)
}

func equalEps(expected, actual float64) bool {
	return math.Abs(expected-actual) < eps
}

func assertErr(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func assertErrIs(t *testing.T, err, target error) {
	t.Helper()

	if errors.Is(err, target) {
		return
	}

	t.Errorf("errors don't match, err: %s, target err: %s", err, target)
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, expected, actual T, msgAndArgs ...any) {
	t.Helper()

	if expected != actual {
		var (
			expStr, actStr string

			e, a any = expected, actual
		)

		switch e.(type) {
		case Cell:
			eC, _ := e.(Cell)
			aC, _ := a.(Cell)

			expStr = fmt.Sprintf("%s (res=%d)", eC, eC.Resolution())
			actStr = fmt.Sprintf("%s (res=%d)", aC, aC.Resolution())
		default:
			expStr = fmt.Sprintf("%v", e)
			actStr = fmt.Sprintf("%v", a)
		}

		t.Errorf("%v != %v", expStr, actStr)
		logMsgAndArgs(t, msgAndArgs...)
	}
}

func assertEqualEps(t *testing.T, expected, actual float64, msgAndArgs ...any) {
	t.Helper()

	if !equalEps(expected, actual) {
		t.Errorf("%0.4f > %v (%0.4f - %0.4f)", math.Abs(expected-actual), eps, expected, actual)
		logMsgAndArgs(t, msgAndArgs...)
	}
}

func assertEqualLatLng(t *testing.T, expected, actual LatLng) {
	t.Helper()
	assertEqualEps(t, expected.Lat, actual.Lat, "latitude mismatch")
	assertEqualEps(t, expected.Lng, actual.Lng, "longitude mismatch")
}

func assertEqualLatLngs(t *testing.T, expected, actual []LatLng, msgAndArgs ...any) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("length mismatch: %v != %v", len(expected), len(actual))
		logMsgAndArgs(t, msgAndArgs...)

		return
	}

	count := 0

	for i, ll := range expected {
		equalLat := equalEps(ll.Lat, actual[i].Lat)
		equalLng := equalEps(ll.Lng, actual[i].Lng)

		if !equalLat || !equalLng {
			latStr := tern(equalLat, fmt.Sprintf("%v", ll.Lat), fmt.Sprintf("%v != %v", ll.Lat, actual[i].Lat))
			lngStr := tern(equalLng, fmt.Sprintf("%v", ll.Lng), fmt.Sprintf("%v != %v", ll.Lng, actual[i].Lng))

			t.Errorf("LatLngs[%d]: (%s, %s)", i, latStr, lngStr)
			logMsgAndArgs(t, msgAndArgs...)

			count++

			if count > 10 {
				t.Log("...and more")
				break
			}
		}
	}
}

func assertEqualCells(t *testing.T, expected, actual []Cell, msgAndArgs ...any) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("length mismatch: %v != %v", len(expected), len(actual))
		logMsgAndArgs(t, msgAndArgs...)

		return
	}

	expected = sortCells(copyCells(expected))
	actual = sortCells(copyCells(actual))

	count := 0

	for i, c := range expected {
		if c != actual[i] {
			t.Errorf("Cells[%d]: %v != %v", i, c, actual[i])
			logMsgAndArgs(t, msgAndArgs...)

			count++

			if count > 10 {
				t.Log("...and more")
				break
			}
		}
	}
}

func assertEqualDiskDistances(t *testing.T, expected, actual [][]Cell) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("number of rings mismatch: %v != %v", len(expected), len(actual))
		return
	}

	expected = copyRings(expected)
	actual = copyRings(actual)

	for i := range expected {
		if len(expected[i]) != len(actual[i]) {
			t.Errorf("ring[%d] length mismatch: %v != %v", i, len(expected[i]), len(actual[i]))
			return
		}

		expected[i] = sortCells(expected[i])
		actual[i] = sortCells(actual[i])

		for j, cell := range expected[i] {
			if cell != actual[i][j] {
				t.Errorf("ring[%d][%d] mismatch: %v != %v", i, j, cell, actual[i][j])
				return
			}
		}
	}
}

func assertEqualDisks(t *testing.T, expected, actual []Cell, msgAndArgs ...any) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("disk size mismatch: %v != %v", len(expected), len(actual))
		logMsgAndArgs(t, msgAndArgs...)

		return
	}

	expected = sortCells(copyCells(expected))
	actual = sortCells(copyCells(actual))

	count := 0

	for i, cell := range expected {
		if cell != actual[i] {
			t.Errorf("cell[%d]: %v != %v", i, cell, actual[i])
			logMsgAndArgs(t, msgAndArgs...)

			count++
			if count > 5 {
				t.Logf("... and more")
				break
			}
		}
	}
}

func assertCellIn(t *testing.T, needle Cell, haystack []Cell) {
	t.Helper()

	var found bool
	for _, h := range haystack {
		found = needle == h
		if found {
			break
		}
	}

	if !found {
		t.Errorf("%v not found in %+v", needle, haystack)
	}
}

func assertNoPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()

	f()
}

func assertFalse(t *testing.T, b bool) {
	t.Helper()
	assertEqual(t, false, b)
}

func assertTrue(t *testing.T, b bool) {
	t.Helper()
	assertEqual(t, true, b)
}

func assertNil(t *testing.T, val any) {
	t.Helper()

	if val == nil {
		return
	}

	value := reflect.ValueOf(val)
	switch value.Kind() {
	case
		reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		if value.IsNil() {
			return
		}
	default:
		t.Errorf("expected value to be nil, got: %v", val)
		return
	}

	t.Errorf("expected value to be nil, got: %v", val)
}

func sortCells(s []Cell) []Cell {
	sort.SliceStable(s, func(i, j int) bool {
		return s[i] < s[j]
	})

	return s
}

func logMsgAndArgs(t *testing.T, msgAndArgs ...any) {
	t.Helper()

	if len(msgAndArgs) > 0 {
		format, _ := msgAndArgs[0].(string)
		t.Logf(format, msgAndArgs[1:]...)
	}
}

func flattenDisks(diskDist [][]Cell) []Cell {
	if len(diskDist) == 0 {
		return nil
	}

	flat := make([]Cell, 0, maxGridDiskSize(len(diskDist)-1))
	for _, disk := range diskDist {
		flat = append(flat, disk...)
	}

	return flat
}

func tern(b bool, x, y string) string {
	if b {
		return x
	}

	return y
}

func copyRings(s [][]Cell) [][]Cell {
	c := make([][]Cell, len(s))
	copy(c, s)

	for i := range c {
		c[i] = append([]Cell{}, s[i]...)
	}

	return c
}

func copyCells(s []Cell) []Cell {
	c := make([]Cell, len(s))
	copy(c, s)

	return c
}

func TestToErr(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assertNoErr(t, toErr(0))
	})

	t.Run("pentagon error", func(t *testing.T) {
		t.Parallel()
		assertErrIs(t, toErr(9), ErrPentagon)
	})

	t.Run("unknown error", func(t *testing.T) {
		t.Parallel()
		assertErrIs(t, toErr(999), ErrFailed)
	})
}

func TestLatLngsToC_Nil(t *testing.T) {
	assertEqual(t, nil, latLngsToC(nil))
}

func TestLatLng_String(t *testing.T) {
	t.Parallel()

	assertEqual(t, "(67.15093, -168.39089)", validLatLng1.String())
}

func TestIndexDigit(t *testing.T) {
	t.Run("cell", func(t *testing.T) {
		indexDigit, err := validCell.IndexDigit(2)
		assertEqual(t, 5, indexDigit)
		assertNoErr(t, err)
	})
	t.Run("edge", func(t *testing.T) {
		indexDigit, err := validEdge.IndexDigit(2)
		assertEqual(t, 5, indexDigit)
		assertNoErr(t, err)
	})
	t.Run("vertex", func(t *testing.T) {
		indexDigit, err := validVertex.IndexDigit(2)
		assertEqual(t, 5, indexDigit)
		assertNoErr(t, err)
	})
	t.Run("err/invalid_res", func(t *testing.T) {
		_, err := validVertex.IndexDigit(-1)
		assertErrIs(t, err, ErrResolutionDomain)
	})
}

func TestIsValidIndex(t *testing.T) {
	testCases := []struct {
		name    string
		isValid bool
		input   any
	}{
		{name: "valid cell", isValid: true, input: validCell},
		{name: "valid vertex", isValid: true, input: validVertex},
		{name: "valid edge", isValid: true, input: validEdge},
		{name: "invalid cell", isValid: false, input: Cell(0)},
		{name: "invalid vertex", isValid: false, input: Vertex(0)},
		{name: "invalid edge", isValid: false, input: DirectedEdge(0)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result bool

			switch v := tc.input.(type) {
			case Cell:
				result = IsValidIndex(v)
			case Vertex:
				result = IsValidIndex(v)
			case DirectedEdge:
				result = IsValidIndex(v)
			default:
				t.Errorf("unexpected input type, input: %v", tc.input)
			}

			assertEqual(t, tc.isValid, result)
		})
	}
}
