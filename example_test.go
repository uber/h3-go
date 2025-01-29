package h3_test

import (
	"fmt"

	"github.com/bulatsan/h3-go/v4"
)

func ExampleLatLngToCell() {
	latLng := h3.NewLatLng(37.775938728915946, -122.41795063018799)
	resolution := 9

	c, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", c)
	// Output:
	// 8928308280fffff
}
