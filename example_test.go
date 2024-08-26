package h3_test

import (
	"fmt"

	"github.com/uber/h3-go/v4"
)

func ExampleLatLngToCell() {
	latLng := h3.NewLatLng(37.775938728915946, -122.41795063018799)
	resolution := 9

	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		fmt.Printf("%s", err.Error())
	} else {
		fmt.Printf("%s", cell)
	}

	// Output:
	// 8928308280fffff
}
