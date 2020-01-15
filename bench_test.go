package h3

import (
	"testing"
)

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

func BenchmarkPolyfill(b *testing.B) {
	for n := 0; n < b.N; n++ {
		h3idxs = Polyfill(validGeopolygonWithHoles, 6)
	}
}

var (
	hexes           [][]H3Index
	hexRangesCenter = H3Index(0x8928308280fffff)
	hexRangeK       = 5
)

func BenchmarkHexRangesNative(b *testing.B) {
	group := KRing(hexRangesCenter, hexRangeK)
	for n := 0; n < b.N; n++ {
		hexes = make([][]H3Index, len(group))
		for i, originHex := range group {
			out, _ := HexRange(originHex, hexRangeK)
			hexes[i] = out
		}
	}
}

func BenchmarkHexRangesC(b *testing.B) {
	group := KRing(hexRangesCenter, hexRangeK)
	for n := 0; n < b.N; n++ {
		hexes, _ = HexRanges(group, hexRangeK)
	}
}
