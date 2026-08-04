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

// LatLngToCell returns the Cell at resolution for a geographic coordinate.
func LatLngToCell(latLng LatLng, res int) (Cell, error) {
	if res < 0 || res > maxResolution {
		return 0, ErrResolutionDomain
	}

	if math.IsNaN(latLng.Lat) || math.IsInf(latLng.Lat, 0) ||
		math.IsNaN(latLng.Lng) || math.IsInf(latLng.Lng, 0) {
		return 0, ErrLatLngDomain
	}

	lat := latLng.Lat * degsToRads
	lng := latLng.Lng * degsToRads
	v := latLngToVec3(lat, lng)
	fijk := v.toFaceIjk(res)

	return fijk.toH3(res)
}

// CellToLatLng returns the geographic center point of a cell in degrees.
func CellToLatLng(c Cell) (LatLng, error) {
	return c.LatLng()
}

// LatLng returns the geographic center point of the cell in degrees.
func (c Cell) LatLng() (LatLng, error) {
	fijk, err := c.toFaceIjk()
	if err != nil {
		return LatLng{}, err
	}

	return fijk.toVec3(c.Resolution()).toLatLng(), nil
}

// --- Vec3d math ---

func latLngToVec3(lat, lng float64) vec3d {
	r := math.Cos(lat)

	return vec3d{
		x: math.Cos(lng) * r,
		y: math.Sin(lng) * r,
		z: math.Sin(lat),
	}
}

// toLatLng converts a 3D unit vector on the sphere into a geographic coordinate
// in degrees.
func (v vec3d) toLatLng() LatLng {
	return LatLng{
		Lat: radsToDegs * math.Asin(v.z),
		Lng: radsToDegs * math.Atan2(v.y, v.x),
	}
}

func (v vec3d) dot(b vec3d) float64 {
	return v.x*b.x + v.y*b.y + v.z*b.z
}

func (v vec3d) cross(b vec3d) vec3d {
	return vec3d{
		x: v.y*b.z - v.z*b.y,
		y: v.z*b.x - v.x*b.z,
		z: v.x*b.y - v.y*b.x,
	}
}

// linComb returns the linear combination scaleA·v + scaleB·other.
func (v vec3d) linComb(scaleA, scaleB float64, other vec3d) vec3d {
	return vec3d{
		x: scaleA*v.x + scaleB*other.x,
		y: scaleA*v.y + scaleB*other.y,
		z: scaleA*v.z + scaleB*other.z,
	}
}

func (v *vec3d) normalize() {
	sq := v.dot(*v)

	s := 0.0
	if sq > 0 {
		s = 1.0 / math.Sqrt(sq)
	}

	v.x *= s
	v.y *= s
	v.z *= s
}

func (v vec3d) distSq(b vec3d) float64 {
	d := v.linComb(1.0, -1.0, b)

	return d.dot(d)
}
