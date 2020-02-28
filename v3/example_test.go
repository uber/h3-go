package h3

import (
	"fmt"
)

func ExampleFromGeo() {
	geo := GeoCoord{
		Latitude:  37.775938728915946,
		Longitude: -122.41795063018799,
	}
	resolution := 9
	fmt.Printf("%#x\n", FromGeo(geo, resolution))
	// Output:
	// 0x8928308280fffff
}
