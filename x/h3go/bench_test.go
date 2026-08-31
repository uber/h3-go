package h3go

import (
	"strconv"
	"testing"
)

// benchResolutions spans both the Class II (even) and Class III (odd) code
// paths, from the coarsest to the finest resolution.
var benchResolutions = []int{0, 2, 7, 9, 15}

// benchPoint is a fixed mid-latitude coordinate that projects onto a hexagon
// (not a face center, pole, or pentagon), so it exercises the common path.
var benchPoint = LatLng{Lat: 37.7749, Lng: -122.4194}

// Hierarchy benchmarks expand a parent into its children; these bounds keep the
// hexagon fan-out at 7^(7-3) = 2401 cells per op.
const (
	benchParentRes = 3
	benchChildRes  = 7
)

// Result sinks
var (
	sinkCell     Cell
	sinkLatLng   LatLng
	sinkBoundary CellBoundary
	sinkCells    []Cell
	sinkInt      int
	sinkStr      string
	sinkBool     bool
)

// benchHexCell derives the hexagon cell covering benchPoint at res.
func benchHexCell(tb testing.TB, res int) Cell {
	tb.Helper()

	c, err := LatLngToCell(benchPoint, res)
	if err != nil {
		tb.Fatalf("LatLngToCell(%v, %d): %v", benchPoint, res, err)
	}

	return c
}

// benchPentCell returns the first pentagon at res.
func benchPentCell(tb testing.TB, res int) Cell {
	tb.Helper()

	pents, err := Pentagons(res)
	if err != nil || len(pents) == 0 {
		tb.Fatalf("Pentagons(%d): %v", res, err)
	}

	return pents[0]
}

// Forward/reverse projections

func BenchmarkLatLngToCell(b *testing.B) {
	for _, res := range benchResolutions {
		b.Run(strconv.Itoa(res), func(b *testing.B) {
			var c Cell
			for b.Loop() {
				c, _ = LatLngToCell(benchPoint, res)
			}
			sinkCell = c
		})
	}
}

func BenchmarkCellToLatLng(b *testing.B) {
	for _, res := range benchResolutions {
		c := benchHexCell(b, res)
		b.Run(strconv.Itoa(res), func(b *testing.B) {
			var ll LatLng
			for b.Loop() {
				ll, _ = CellToLatLng(c)
			}
			sinkLatLng = ll
		})
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for _, res := range benchResolutions {
		hex := benchHexCell(b, res)
		pent := benchPentCell(b, res)

		b.Run("hex/"+strconv.Itoa(res), func(b *testing.B) {
			b.ReportAllocs()

			var bnd CellBoundary
			for b.Loop() {
				bnd, _ = CellToBoundary(hex)
			}
			sinkBoundary = bnd
		})

		b.Run("pent/"+strconv.Itoa(res), func(b *testing.B) {
			b.ReportAllocs()

			var bnd CellBoundary
			for b.Loop() {
				bnd, _ = CellToBoundary(pent)
			}
			sinkBoundary = bnd
		})
	}
}

// String conversion

func BenchmarkCellFromString(b *testing.B) {
	s := benchHexCell(b, maxResolution).String()

	var c Cell
	for b.Loop() {
		c = CellFromString(s)
	}
	sinkCell = c
}

func BenchmarkCellToString(b *testing.B) {
	c := benchHexCell(b, maxResolution)

	var s string
	for b.Loop() {
		s = c.String()
	}
	sinkStr = s
}

// Introspection

func BenchmarkIsValid(b *testing.B) {
	c := benchHexCell(b, maxResolution)

	var ok bool
	for b.Loop() {
		ok = c.IsValid()
	}
	sinkBool = ok
}

func BenchmarkIsPentagon(b *testing.B) {
	c := benchPentCell(b, maxResolution)

	var ok bool
	for b.Loop() {
		ok = c.IsPentagon()
	}
	sinkBool = ok
}

// Hierarchy/set operations

func BenchmarkParent(b *testing.B) {
	c := benchHexCell(b, maxResolution)

	var p Cell
	for b.Loop() {
		p, _ = c.Parent(benchParentRes)
	}
	sinkCell = p
}

func BenchmarkChildPos(b *testing.B) {
	c := benchHexCell(b, 12)

	var pos int
	for b.Loop() {
		pos, _ = c.ChildPos(benchParentRes)
	}
	sinkInt = pos
}

func BenchmarkChildren(b *testing.B) {
	hex := benchHexCell(b, benchParentRes)
	pent := benchPentCell(b, benchParentRes)

	b.Run("hex", func(b *testing.B) {
		b.ReportAllocs()

		var out []Cell
		for b.Loop() {
			out, _ = hex.Children(benchChildRes)
		}
		sinkCells = out
	})

	b.Run("pent", func(b *testing.B) {
		b.ReportAllocs()

		var out []Cell
		for b.Loop() {
			out, _ = pent.Children(benchChildRes)
		}
		sinkCells = out
	})
}

func BenchmarkUncompactCells(b *testing.B) {
	in := []Cell{benchHexCell(b, benchParentRes)}

	b.ReportAllocs()

	var out []Cell
	for b.Loop() {
		out, _ = UncompactCells(in, benchChildRes)
	}
	sinkCells = out
}

func BenchmarkCompactCells(b *testing.B) {
	full, err := benchHexCell(b, benchParentRes).Children(benchChildRes)
	if err != nil {
		b.Fatalf("Children: %v", err)
	}

	b.ReportAllocs()

	var out []Cell
	for b.Loop() {
		out, _ = CompactCells(full)
	}
	sinkCells = out
}
