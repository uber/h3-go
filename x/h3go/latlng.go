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
	fijk := vec3ToFaceIjk(v, res)

	return faceIjkToH3(fijk, res)
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

func vec3Dot(a, b vec3d) float64 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func vec3Cross(a, b vec3d) vec3d {
	return vec3d{
		x: a.y*b.z - a.z*b.y,
		y: a.z*b.x - a.x*b.z,
		z: a.x*b.y - a.y*b.x,
	}
}

func vec3LinComb(a float64, v1 vec3d, b float64, v2 vec3d) vec3d {
	return vec3d{
		x: a*v1.x + b*v2.x,
		y: a*v1.y + b*v2.y,
		z: a*v1.z + b*v2.z,
	}
}

func vec3Normalize(v *vec3d) {
	sq := vec3Dot(*v, *v)

	s := 0.0
	if sq > 0 {
		s = 1.0 / math.Sqrt(sq)
	}

	v.x *= s
	v.y *= s
	v.z *= s
}

func vec3DistSq(a, b vec3d) float64 {
	d := vec3LinComb(1.0, a, -1.0, b)

	return vec3Dot(d, d)
}
