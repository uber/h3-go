<img align="right" src="https://uber.github.io/img/h3Logo-color.svg" alt="H3 Logo" width="200">

![Build](https://github.com/uber/h3-go/workflows/Build/badge.svg?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/uber/h3-go/badge.svg)](https://coveralls.io/github/uber/h3-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/uber/h3-go)](https://goreportcard.com/report/github.com/uber/h3-go)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![GoDoc](http://img.shields.io/badge/go-doc-blue.svg)](https://godoc.org/github.com/uber/h3-go)
[![H3 Version](https://img.shields.io/badge/h3-v3.7.0-blue.svg)](https://github.com/uber/h3/releases/tag/v3.7.0)

<h1 align="center">H3-Go is looking for a maintainer familiar with Go, C, and H3. Volunteers welcome! Please get in touch with us on the <a href="https://join.slack.com/t/h3-core/shared_invite/zt-g6u5r1hf-W_~uVJmfeiWtMQuBGc1NNg">H3 Slack</a>.</h1>

# H3-Go

This library provides Golang bindings for the
[H3 Core Library](https://github.com/uber/h3). For API reference, see the
[H3 Documentation](https://uber.github.io/h3/).

**This is v4 of this library, supporting H3 v4.**

Check out [v3](https://github.com/uber/h3-go/tree/v3.7.1) or checkout the git tag for the version you need.

**Migrating from v3?**

Check out [v3 to v4 migration guide](https://h3geo.org/docs/library/migrating-3.x).
There have been no breaking changes to the format of H3 indexes.  Indexes
generated by older versions can be parsed in v4, and vice-versa.

# Usage

## Prerequisites

H3-Go requires [CGO](https://golang.org/cmd/cgo/) (`CGO_ENABLED=1`) in order to
be built. Go should do the right thing when including this library:

> The cgo tool is enabled by default for native builds on systems where it is
> expected to work. It is disabled by default when cross-compiling. You can
> control this by setting the CGO_ENABLED environment variable when running the go
> tool: set it to 1 to enable the use of cgo, and to 0 to disable it. The go tool
> will set the build constraint "cgo" if cgo is enabled. The special import "C"
> implies the "cgo" build constraint, as though the file also said "// +build
> cgo". Therefore, if cgo is disabled, files that import "C" will not be built by
> the go tool. (For more about build constraints see
> <https://golang.org/pkg/go/build/#hdr-Build_Constraints>).

If you see errors/warnings like _"build constraints exclude all Go files..."_,
then the `cgo` build constraint is likely disabled; try setting `CGO_ENABLED=1`
environment variable in your `go build` step.

## Installation

```bash
go get github.com/uber/h3-go/v4
```

## Quickstart

```go
import "github.com/uber/h3-go/v4"

func ExampleLatLngToCell() {
 latLng := h3.NewLatLng(37.775938728915946, -122.41795063018799)
 resolution := 9 // between 0 (biggest cell) and 15 (smallest cell)

 cell := h3.LatLngToCell(latLng, resolution)

 fmt.Printf("%s", cell)
 // Output:
 // 8928308280fffff
}

```

# C API

## Notes

* `LatLng` returns `Lat` and `Lng` as degrees, instead of radians.
* H3 C API function prefixes of `get` have been dropped in support of Golang's
 `Getter` [naming style](https://golang.org/doc/effective_go.html#Getters).
* Convenience methods have been added to various types where that type was the
  main or only argument.

## Bindings

| C API                        | Go API                                             |
| ---------------------------- |----------------------------------------------------|
| `latLngToCell`               | `LatLngToCell`, `LatLng#Cell`                      |
| `cellToLatLng`               | `CellToLatLng`, `Cell#LatLng`                      |
| `cellToBoundary`             | `CellToBoundary`, `Cell#Boundary`                  |
| `gridDisk`                   | `GridDisk`, `Cell#GridDisk`                        |
| `gridDiskDistances`          | `GridDiskDistances`, `Cell#GridDiskDistances`      |
| `gridRingUnsafe`             | N/A                                                |
| `polygonToCells`             | `PolygonToCells`, `GeoPolygon#Cells`               |
| `cellsToMultiPolygon`        | `CellsToMultiPolygon`                               |
| `degsToRads`                 | `DegsToRads`                                       |
| `radsToDegs`                 | `RadsToDegs`                                       |
| `greatCircleDistance`        | `GreatCircleDistance* (3/3)`                       |
| `getHexagonAreaAvg`          | `HexagonAreaAvg* (3/3)`                            |
| `cellArea`                   | `CellArea* (3/3)`                                  |
| `getHexagonEdgeLengthAvg`    | `HexagonEdgeLengthAvg* (2/2)`                      |
| `exactEdgeLength`            | `EdgeLength* (3/3)`                                |
| `getNumCells`                | `NumCells`                                         |
| `getRes0Cells`               | `Res0Cells`                                        |
| `getPentagons`               | `Pentagons`                                        |
| `getResolution`              | `Resolution`                                       |
| `getBaseCellNumber`          | `BaseCellNumber`, `Cell#BaseCellNumber`            |
| `stringToH3`                 | `IndexFromString`, `Cell#UnmarshalText`            |
| `h3ToString`                 | `IndexToString`, `Cell#String`, `Cell#MarshalText` |
| `isValidCell`                | `Cell#IsValid`                                     |
| `cellToParent`               | `Cell#Parent`, `Cell#ImmediateParent`              |
| `cellToChildren`             | `Cell#Children` `Cell#ImmediateChildren`           |
| `cellToCenterChild`          | `Cell#CenterChild`                                 |
| `compactCells`               | `CompactCells`                                     |
| `uncompactCells`             | `UncompactCells`                                   |
| `isResClassIII`              | `Cell#IsResClassIII`                               |
| `isPentagon`                 | `Cell#IsPentagon`                                  |
| `getIcosahedronFaces`        | `Cell#IcosahedronFaces`                            |
| `areNeighborCells`           | `Cell#IsNeighbor`                                  |
| `cellsToDirectedEdge`        | `Cell#DirectedEdge`                                |
| `isValidDirectedEdge`        | `DirectedEdge#IsValid`                             |
| `getDirectedEdgeOrigin`      | `DirectedEdge#Origin`                              |
| `getDirectedEdgeDestination` | `DirectedEdge#Destination`                         |
| `directedEdgeToCells`        | `DirectedEdge#Cells`                               |
| `originToDirectedEdges`      | `Cell#DirectedEdges`                               |
| `directedEdgeToBoundary`     | `DirectedEdge#Boundary`                            |
| `cellToVertex`               | TODO                                               |
| `cellToVertexes`             | TODO                                               |
| `vertexToLatLng`             | TODO                                               |
| `isValidVertex`              | TODO                                               |
| `gridDistance`               | `GridDistance`, `Cell#GridDistance`                |
| `gridPathCells`              | `GridPath`, `Cell#GridPath`                        |
| `cellToLocalIj`              | `CellToLocalIJ`                                    |
| `localIjToCell`              | `LocalIJToCell`                                    |

## CGO

The H3 C source code and header files are copied into this project to optimize
for portability. `h3-go` can be imported into any Go project for any platform
that CGO supports.

# Contributing

Pull requests and Github issues are welcome.  Please read our [contributing
guide](./CONTRIBUTING.md) for more information.

## Legal and Licensing

H3-Go is licensed under the [Apache 2.0 License](./LICENSE).
