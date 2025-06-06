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
	disks    [][]Cell
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

func BenchmarkCellToLatLng(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geo, _ = CellToLatLng(cell)
	}
}

func BenchmarkLatLngToCell(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cell, _ = LatLngToCell(geo, 15)
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geoBndry, _ = CellToBoundary(cell)
	}
}

func BenchmarkGridDisk(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cells, _ = cell.GridDisk(10)
	}
}

func BenchmarkGridRing(b *testing.B) {
	for range b.N {
		cells, _ = cell.GridRing(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cells, _ = PolygonToCells(validGeoPolygonHoles, 13)
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	cells, _ = PolygonToCells(validGeoPolygonHoles, 12)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		disks, _ = GridDisksUnsafe(cells, 10)
	}
}
