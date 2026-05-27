package h3

import (
	"strconv"
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
	latlngStr  string
	cell, _    = LatLngToCell(geo, 15)
	addr       = cell.String()
	geoBndry   CellBoundary
	cells      []Cell
	disks      [][]Cell
	distResult float64
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

func BenchmarkLatLngToCellsBatch(b *testing.B) {
	for _, n := range []int{1, 64, 1024, 16384, 1_000_000, 10_000_000} {
		lls := make([]LatLng, n)
		for i := range lls {
			lls[i] = LatLng{Lat: 37.7749 + float64(i)*1e-6, Lng: -122.4194}
		}

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			for b.Loop() {
				cells, _ := LatLngToCellsBatch(lls, 9)
				sink = int(cells[0])
			}
		})
	}
}

// Baseline: same workload via the per-call LatLngToCell in a Go loop,
// so reviewers can confirm the speedup at each batch size by comparing
// matching sub-benchmark names with benchstat.
func BenchmarkLatLngToCellsBaseline(b *testing.B) {
	for _, n := range []int{1, 64, 1024, 16384, 1_000_000, 10_000_000} {
		lls := make([]LatLng, n)
		for i := range lls {
			lls[i] = LatLng{Lat: 37.7749 + float64(i)*1e-6, Lng: -122.4194}
		}

		b.Run(strconv.Itoa(n), func(b *testing.B) {
			for range b.N {
				for _, ll := range lls {
					cell, _ = LatLngToCell(ll, 9)
				}
			}
		})
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

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	for range b.N {
		distResult = GreatCircleDistanceRads(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	for range b.N {
		distResult = GreatCircleDistanceKm(geo, geo2)
	}
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	for range b.N {
		distResult = GreatCircleDistanceM(geo, geo2)
	}
}
