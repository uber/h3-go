# H3-Go

This library provides Golang bindings for the [H3 Core
Library](https://github.com/uber/h3). For API reference, please see the [H3
Documentation](https://uber.github.io/h3/).

# Usage

## Installation

### [golang/dep](https://github.com/golang/dep)

```bash
dep ensure -add github.com/uber/h3-go
```

### [golang/cmd/go](https://golang.org/cmd/go/)

```bash
go get github.com/uber/h3-go
```

### [Glide](https://github.com/Masterminds/glide)

```bash
glide install github.com/uber/h3-go
```

## Quickstart

```go
import "github.com/uber/h3-go"

func ExampleFromGeo() {
	geo := h3.GeoCoord{
		Latitude:  37.775938728915946,
		Longitude: -122.41795063018799,
	}
	resolution := 9
	fmt.Printf("%#x\n", h3.FromGeo(geo, resolution))
	// Output:
	// 0x8928308280fffff
}
```

# Notes

## API Differences

Some superficial changes have been made relative to the H3 C core API in order
to adhere to idiomatic Go styling.  Most notable are the following:

* H3 C API function prefixes of `H3` have been dropped to reduce stutter in
  usage, e.g. `h3.ToGeo(h)`.
* H3 C functions that convert **to** `H3Index` have their names inverted to
  convert **from** something else to `H3Index`, e.g. `GeoToH3` is renamed to
  `h3.FromGeo`.
* H3 C API function prefixes of `Get` have been dropped in support of Golang's
  `Getter` [naming style](https://golang.org/doc/effective_go.html#Getters).

## CGO

The H3 C source code and header files are copied into this project to optimize
for portability.  By including the C source files in the `h3` Go package, there
is no need to introduce a build process or a system dependency on an H3 binary.
Effectively, this decision makes `h3` as easy to use in a Go project as adding
it as a dependency with your favorite dependency manager.

# Contributing

Pull requests and Github issues are welcome.  Please read our [contributing
guide](./CONTRIBUTING.md) for more information.

## Legal and Licensing

H3-Go is licensed under the [Apache 2.0 License](./LICENSE).
