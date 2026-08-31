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
	geo2 = LatLng{
		Lat: 38,
		Lng: -121,
	}
	latlngStr   string
	cell, _     = LatLngToCell(geo, 15)
	addr        = cell.String()
	geoBndry    CellBoundary
	cells       []Cell
	disks       [][]Cell
	distResult  float64
	boolResult  bool
	intResult   int
	validResult bool
)

func BenchmarkToString(b *testing.B) {
	for b.Loop() {
		addr = cell.String()
	}
}

func BenchmarkFromString(b *testing.B) {
	for b.Loop() {
		cell = Cell(IndexFromString("850dab63fffffff"))
	}
}

func BenchmarkLatLng_String(b *testing.B) {
	for b.Loop() {
		latlngStr = geo.String()
	}
}

func BenchmarkCellToLatLng(b *testing.B) {
	for b.Loop() {
		geo, _ = CellToLatLng(cell)
	}
}

func BenchmarkLatLngToCell(b *testing.B) {
	for b.Loop() {
		cell, _ = LatLngToCell(geo, 15)
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for b.Loop() {
		geoBndry, _ = CellToBoundary(cell)
	}
}

func BenchmarkGridDisk(b *testing.B) {
	for b.Loop() {
		cells, _ = cell.GridDisk(10)
	}
}

func BenchmarkGridRing(b *testing.B) {
	for b.Loop() {
		cells, _ = cell.GridRing(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for b.Loop() {
		cells, _ = PolygonToCells(validGeoPolygonHoles, 13)
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	cells, _ = PolygonToCells(validGeoPolygonHoles, 12)

	

	for b.Loop() {
		disks, _ = GridDisksUnsafe(cells, 10)
	}
}

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	for b.Loop() {
		distResult = GreatCircleDistanceRads(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	for b.Loop() {
		distResult = GreatCircleDistanceKm(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	for b.Loop() {
		distResult = GreatCircleDistanceM(geo, geo2)
	}
}

func BenchmarkResolution(b *testing.B) {
	for b.Loop() {
		intResult = cell.Resolution()
	}
}

func BenchmarkBaseCellNumber(b *testing.B) {
	for b.Loop() {
		intResult = cell.BaseCellNumber()
	}
}

func BenchmarkIsValid(b *testing.B) {
	for b.Loop() {
		boolResult = cell.IsValid()
	}
}

func BenchmarkIsPentagon(b *testing.B) {
	for b.Loop() {
		boolResult = cell.IsPentagon()
	}
}

func BenchmarkIsResClassIII(b *testing.B) {
	for b.Loop() {
		boolResult = cell.IsResClassIII()
	}
}

func BenchmarkIsValidIndex(b *testing.B) {
	for b.Loop() {
		validResult = IsValidIndex(cell)
	}
}
