/*
 * Copyright 2026 Uber Technologies, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package h3go

import "math"

// CellAreaRads2 returns the exact area of a cell in square radians.
func CellAreaRads2(c Cell) (float64, error) {
	boundary, err := c.Boundary()
	if err != nil {
		return 0, err
	}

	return boundary.areaRads2(), nil
}

// CellAreaKm2 returns the exact area of a cell in square kilometers.
func CellAreaKm2(c Cell) (float64, error) {
	rads2, err := CellAreaRads2(c)
	if err != nil {
		return 0, err
	}

	return rads2 * earthRadiusKm * earthRadiusKm, nil
}

// CellAreaM2 returns the exact area of a cell in square meters.
func CellAreaM2(c Cell) (float64, error) {
	km2, err := CellAreaKm2(c)
	if err != nil {
		return 0, err
	}

	return km2 * 1000 * 1000, nil
}

// areaRads2 returns the area in square radians enclosed by the boundary loop. It
// sums the signed Cagnoli area contribution of each edge arc (assumed to be the
// shorter geodesic) with compensated summation, then normalizes a clockwise loop
// into [0, 4π] by adding the full-sphere area.
//
// Every vertex bounds two edges, so its trig terms are evaluated once and
// carried from one edge to the next rather than recomputed per edge.
func (b CellBoundary) areaRads2() float64 {
	if len(b) == 0 {
		return 0
	}

	var adder kahanAdder

	first := b[0].cagnoliVertex()
	prev := first

	for i := 1; i < len(b); i++ {
		cur := b[i].cagnoliVertex()
		adder.add(prev.cagnoli(cur))
		prev = cur
	}

	adder.add(prev.cagnoli(first))

	if adder.sum < 0 {
		adder.add(2 * m2PI) // 4π, the area of the whole sphere
	}

	return adder.sum
}

// cagnoliVertex holds the per-vertex terms of the Cagnoli edge formula: the
// sine and cosine of lat/2 + π/4, and the longitude in degrees.
type cagnoliVertex struct {
	sinHalfLat float64
	cosHalfLat float64
	lng        float64
}

// cagnoliVertex evaluates the per-vertex Cagnoli terms of a point given in
// degrees.
func (ll LatLng) cagnoliVertex() cagnoliVertex {
	halfLat := ll.Lat*DegsToRads/2 + math.Pi/4

	return cagnoliVertex{
		sinHalfLat: math.Sin(halfLat),
		cosHalfLat: math.Cos(halfLat),
		lng:        ll.Lng,
	}
}

// cagnoli returns the signed area contribution, in radians, of the boundary edge
// arc from v to other, following the d3-geo spherical-area formulation.
func (v cagnoliVertex) cagnoli(other cagnoliVertex) float64 {
	sa := v.sinHalfLat * other.sinHalfLat
	ca := v.cosHalfLat * other.cosHalfLat

	delta := (other.lng - v.lng) * DegsToRads
	sinDelta := math.Sin(delta)
	cosDelta := math.Cos(delta)

	return -2 * math.Atan2(sa*sinDelta, sa*cosDelta+ca)
}

// kahanAdder accumulates a sum of floating-point terms using Kahan compensated
// summation, which preserves precision when adding many terms of varying
// magnitude (as a polygon area does).
type kahanAdder struct {
	sum  float64
	comp float64 // running compensation for lost low-order bits
}

// add folds x into the running sum, carrying the rounding error forward.
func (a *kahanAdder) add(x float64) {
	y := x - a.comp
	t := a.sum + y
	a.comp = (t - a.sum) - y
	a.sum = t
}
