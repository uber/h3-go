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
	cell, _  = LatLngToCell(geo, 15)
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
		//nolint:gosec // IndexFromString returns uint64 and fixing that to detect integer overflows will break package API. Let's skip it for now.
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
		cell, _ = LatLngToCell(geo, 15)
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
