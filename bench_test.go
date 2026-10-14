package h3

import (
	"testing"
)

// Inputs at package scope so the compiler cannot constant-fold them into the
// benchmarked call. Results need no sink: b.Loop keeps the arguments and
// results of calls in its body alive.
var (
	geo = LatLng{
		Lat: 37,
		Lng: -122,
	}
	geo2 = LatLng{
		Lat: 38,
		Lng: -121,
	}
	cell, _ = LatLngToCell(geo, 15)
)

func BenchmarkToString(b *testing.B) {
	for b.Loop() {
		_ = cell.String()
	}
}

func BenchmarkFromString(b *testing.B) {
	for b.Loop() {
		IndexFromString("850dab63fffffff")
	}
}

func BenchmarkLatLng_String(b *testing.B) {
	for b.Loop() {
		_ = geo.String()
	}
}

func BenchmarkCellToLatLng(b *testing.B) {
	for b.Loop() {
		_, _ = CellToLatLng(cell)
	}
}

func BenchmarkLatLngToCell(b *testing.B) {
	for b.Loop() {
		_, _ = LatLngToCell(geo, 15)
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for b.Loop() {
		_, _ = CellToBoundary(cell)
	}
}

func BenchmarkGridDisk(b *testing.B) {
	for b.Loop() {
		_, _ = cell.GridDisk(10)
	}
}

func BenchmarkGridRing(b *testing.B) {
	for b.Loop() {
		_, _ = cell.GridRing(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for b.Loop() {
		_, _ = PolygonToCells(validGeoPolygonHoles, 13)
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	cells, _ := PolygonToCells(validGeoPolygonHoles, 12)

	for b.Loop() {
		_, _ = GridDisksUnsafe(cells, 10)
	}
}

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceRads(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceKm(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceM(geo, geo2)
	}
}

func BenchmarkResolution(b *testing.B) {
	for b.Loop() {
		cell.Resolution()
	}
}

func BenchmarkBaseCellNumber(b *testing.B) {
	for b.Loop() {
		cell.BaseCellNumber()
	}
}

func BenchmarkIsValid(b *testing.B) {
	for b.Loop() {
		cell.IsValid()
	}
}

func BenchmarkIsPentagon(b *testing.B) {
	for b.Loop() {
		cell.IsPentagon()
	}
}

func BenchmarkIsResClassIII(b *testing.B) {
	for b.Loop() {
		cell.IsResClassIII()
	}
}

func BenchmarkIsValidIndex(b *testing.B) {
	for b.Loop() {
		IsValidIndex(cell)
	}
}
