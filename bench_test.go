package h3

import (
	"testing"
)

// buckets for preventing compiler optimizing out calls.
var (
	geo = LatLng{
		Lat: 37,
		Lng: -122,
	}
	cell     = LatLngToCell(geo, 15)
	addr     = cell.String()
	geoBndry CellBoundary
	cells    []Cell
)

func BenchmarkToString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		addr = cell.String()
	}
}

func BenchmarkFromString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		//nolint:gosec
		cell = Cell(IndexFromString("850dab63fffffff"))
	}
}

func BenchmarkToGeoRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geo = CellToLatLng(cell)
	}
}

func BenchmarkFromGeoRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cell = LatLngToCell(geo, 15)
	}
}

func BenchmarkToGeoBndryRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geoBndry = CellToBoundary(cell)
	}
}

func BenchmarkHexRange(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cells = cell.GridDisk(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cells = PolygonToCells(validGeoPolygonHoles, 15)
	}
}
