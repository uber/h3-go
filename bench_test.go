package h3

import (
	"math"
	"strconv"
	"testing"
)

// Inputs at package scope so the compiler cannot constant-fold them into the
// benchmarked call. Results need no sink: b.Loop keeps the arguments and
// results of calls in its body alive.
var (
	benchGeo = LatLng{
		Lat: 37,
		Lng: -122,
	}
	benchGeo2 = LatLng{
		Lat: 38,
		Lng: -121,
	}
	benchCell, _ = LatLngToCell(benchGeo, 15)

	// benchPolygon is the boundary of a resolution 5 cell with two holes.
	benchPolygon = GeoPolygon{
		GeoLoop: GeoLoop{
			{Lat: 67.224749856, Lng: -168.523006585},
			{Lat: 67.140938355, Lng: -168.626914333},
			{Lat: 67.067252558, Lng: -168.494913285},
			{Lat: 67.077062918, Lng: -168.259695931},
			{Lat: 67.160561948, Lng: -168.154801171},
			{Lat: 67.234563187, Lng: -168.286102782},
		},
		Holes: []GeoLoop{
			{
				{Lat: 67.2, Lng: -168.4},
				{Lat: 67.1, Lng: -168.4},
				{Lat: 67.1, Lng: -168.3},
				{Lat: 67.2, Lng: -168.3},
			},
			{
				{Lat: 67.21, Lng: -168.41},
				{Lat: 67.22, Lng: -168.41},
				{Lat: 67.22, Lng: -168.42},
			},
		},
	}
)

func BenchmarkToString(b *testing.B) {
	for b.Loop() {
		_ = benchCell.String()
	}
}

func BenchmarkFromString(b *testing.B) {
	for b.Loop() {
		IndexFromString("850dab63fffffff")
	}
}

func BenchmarkLatLng_String(b *testing.B) {
	for b.Loop() {
		_ = benchGeo.String()
	}
}

func BenchmarkCellToLatLng(b *testing.B) {
	for b.Loop() {
		_, _ = CellToLatLng(benchCell)
	}
}

func BenchmarkLatLngToCell(b *testing.B) {
	for b.Loop() {
		_, _ = LatLngToCell(benchGeo, 15)
	}
}

func BenchmarkCellToBoundary(b *testing.B) {
	for b.Loop() {
		_, _ = CellToBoundary(benchCell)
	}
}

func BenchmarkGridDisk(b *testing.B) {
	for b.Loop() {
		_, _ = benchCell.GridDisk(10)
	}
}

func BenchmarkGridRing(b *testing.B) {
	for b.Loop() {
		_, _ = benchCell.GridRing(10)
	}
}

func BenchmarkPolyfill(b *testing.B) {
	for b.Loop() {
		_, _ = PolygonToCells(benchPolygon, 13)
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	cells, _ := PolygonToCells(benchPolygon, 12)

	for b.Loop() {
		_, _ = GridDisksUnsafe(cells, 10)
	}
}

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceRads(benchGeo, benchGeo2)
	}
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceKm(benchGeo, benchGeo2)
	}
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	for b.Loop() {
		GreatCircleDistanceM(benchGeo, benchGeo2)
	}
}

func BenchmarkResolution(b *testing.B) {
	for b.Loop() {
		benchCell.Resolution()
	}
}

func BenchmarkBaseCellNumber(b *testing.B) {
	for b.Loop() {
		benchCell.BaseCellNumber()
	}
}

func BenchmarkIsValid(b *testing.B) {
	for b.Loop() {
		benchCell.IsValid()
	}
}

func BenchmarkIsPentagon(b *testing.B) {
	for b.Loop() {
		benchCell.IsPentagon()
	}
}

func BenchmarkIsResClassIII(b *testing.B) {
	for b.Loop() {
		benchCell.IsResClassIII()
	}
}

func BenchmarkIsValidIndex(b *testing.B) {
	for b.Loop() {
		IsValidIndex(benchCell)
	}
}

func BenchmarkNumCells(b *testing.B) {
	for res := 0; res <= MaxResolution; res++ {
		b.Run(strconv.Itoa(res), func(b *testing.B) {
			for b.Loop() {
				NumCells(res)
			}
		})
	}
}

// Polygons for the polygon-to-cells benchmarks, matching the reference
// library's: a state, a large ellipse and a long thin triangle whose edges are
// far from straight in latitude and longitude.
var (
	// benchColorado is the state of Colorado.
	benchColorado = GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.645778804554910, -1.903190792178713}, {0.645682811446050, -1.780975856637062},
		{0.715595465293187, -1.781167842854781}, {0.715578012000667, -1.903262350678044},
		{0.645778804554910, -1.903190792178713},
	})}

	// benchEllipse is a 100-vertex ellipse about 1800 km by 1100 km, centered
	// at 35°N 95°W with semi-axes of 5° of latitude and 10° of longitude.
	benchEllipse = GeoPolygon{GeoLoop: func() GeoLoop {
		const sides = 100

		loop := make(GeoLoop, sides)
		for i := range loop {
			theta := 2 * math.Pi * float64(i) / sides
			loop[i] = LatLng{Lat: 35 + 5*math.Cos(theta), Lng: -95 + 10*math.Sin(theta)}
		}

		return loop
	}()}

	// benchLondonNY is a sliver between London and New York.
	benchLondonNY = GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.8989737191417, -0.0022305307840}, {0.7105724077059, -1.2916483662309}, {0.8989737191417, -0.0022305307840},
	})}
)

// BenchmarkPolygonToCellsExperimental times the planar and geodesic fills in
// overlapping mode on the three reference polygons.
func BenchmarkPolygonToCellsExperimental(b *testing.B) {
	polygons := []struct {
		name    string
		polygon GeoPolygon
		res     int
	}{
		{"colorado", benchColorado, 5},
		{"ellipse", benchEllipse, 4},
		{"london_ny", benchLondonNY, 3},
	}

	modes := []struct {
		name string
		mode ContainmentMode
	}{
		{"planar", ContainmentOverlapping},
		{"geodesic", ContainmentOverlapping | ContainmentGeodesic},
	}

	for _, pg := range polygons {
		for _, md := range modes {
			b.Run(pg.name+"/"+md.name, func(b *testing.B) {
				for b.Loop() {
					if _, err := PolygonToCellsExperimental(pg.polygon, pg.res, md.mode); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
