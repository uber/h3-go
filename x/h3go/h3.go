// Package h3go is the pure-Go implementation of Uber's H3 Geo Index system.
//
// It is checked against the C reference by the conformance suite in the
// conformance subdirectory, whose README states the contract: discrete
// outputs identical to the reference, coordinates within 1e-12 degrees of
// arc, measures within 1e-9 relative, and knife-edge inputs, such as points
// on cell edges and polygon vertices on cell edges, outside the contract.
// The README also records where this implementation is deliberately
// allowed to differ from the reference.
package h3go
