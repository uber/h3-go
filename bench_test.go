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
	latlngStr string
	cell, _   = LatLngToCell(geo, 15)
	addr      = cell.String()
	geoBndry  CellBoundary
	cells     []Cell
	disks     [][]Cell
)

func BenchmarkToString(b *testing.B) {
	for range b.N {
		addr = cell.String()
	}
}

func BenchmarkFromString(b *testing.B) {
	for range b.N {
		cell = Cell(IndexFromString("850dab63fffffff"))
	}
}

func BenchmarkLatLng_String(b *testing.B) {
	for range b.N {
		latlngStr = geo.String()
	}
}

func BenchmarkCellToLatLng(b *testing.B) {
	for range b.N {
		geo, _ = CellToLatLng(cell)
	}
}

func BenchmarkLatLngToCell(b *testing.B) {
	for range b.N {
		cell, _ = LatLngToCell(geo, 15)
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for range b.N {
		geoBndry, _ = CellToBoundary(cell)
	}
}

func BenchmarkGridDisk(b *testing.B) {
	for range b.N {
		cells, _ = cell.GridDisk(10)
	}
}

func BenchmarkGridRing(b *testing.B) {
	for range b.N {
		cells, _ = cell.GridRing(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for range b.N {
		cells, _ = PolygonToCells(validGeoPolygonHoles, 13)
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	cells, _ = PolygonToCells(validGeoPolygonHoles, 12)

	b.ResetTimer()

	for range b.N {
		disks, _ = GridDisksUnsafe(cells, 10)
	}
}
