package h3

import "testing"

// buckets for preventing compiler optimizing out calls
var (
	geo = GeoCoord{
		Latitude:  37,
		Longitude: -122,
	}
	h3idx    = FromGeo(geo, 15)
	h3addr   = ToString(h3idx)
	geoBndry GeoBoundary
	h3idxs   []H3Index
)

func BenchmarkToString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		h3addr = ToString(h3idx)
	}
}

func BenchmarkFromString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		h3idx = FromString("850dab63fffffff")
	}
}

func BenchmarkToGeoRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geo = ToGeo(h3idx)
	}
}

func BenchmarkFromGeoRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		h3idx = FromGeo(geo, 15)
	}
}

func BenchmarkToGeoBndryRes15(b *testing.B) {
	for n := 0; n < b.N; n++ {
		geoBndry = ToGeoBoundary(h3idx)
	}
}

func BenchmarkHexRange(b *testing.B) {
	for n := 0; n < b.N; n++ {
		h3idxs, _ = HexRange(h3idx, 10)
	}
}
