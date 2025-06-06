# Changelog

This project tracks the **major** and **minor** versions set upstream by
[`h3`](github.com/uber/h3), and introduces backwards-compatible updates and/or
fixes via **patches** with patch version bumps.

**Changelog notes for versions above v4 are under [Releases](https://github.com/uber/h3-go/releases).**

## 4.0.0

All new functions to match H3 v4.

See <https://h3geo.org/docs/library/migrating-3.x> for upstream changes, and the
[README.md](./README.md) for upstream to h3-go binding name mapping.

## 3.7.1

### Added

* Functions to cover full functionality (#46)
  * `Res0IndexCount`
  * `GetRes0Indexes`
  * `DistanceBetween`
  * `ToCenterChild`
  * `MaxFaceCount`
  * `GetFaces`
  * `PentagonIndexCount`
  * `GetPentagonIndexes`
  * `HexAreaKm2`
  * `HexAreaM2`
  * `PointDistRads`
  * `PointDistKm`
  * `PointDistM`
  * `CellAreaRads2`
  * `CellAreaKm2`
  * `CellAreaM2`
  * `EdgeLengthKm`
  * `EdgeLengthM`
  * `ExactEdgeLengthRads`
  * `ExactEdgeLengthKm`
  * `ExactEdgeLengthM`
  * `NumHexagons`

## 3.7.0

### Added

* `SetToLinkedGeo` function (#41)
* `Line` function (#37)

## 3.0.2

### Fixed

* `go mod vendor` now works correctly (#30, #32)

### Added

* Some useful H3 constants (#22):
  * `MaxResolution`
  * `NumIcosaFaces`
  * `NumBaseCells`
* Support for GOMODULES (#24)

## 3.0.1

### Added

* `Polyfill` function (#19).

### Changed

* [breaking] `Uncompat` now returns `([]H3Index, error)` instead of `[]H3Index`
  to accommodate error scenario from C API (#19).

### Fixed

* panic when using `Uncompact` with invalid resolutions (#20).
* latitudes and longitudes outside of respective ranges when unprojecting in
  certain areas (#7, #9, #13).

## v3.0.0

### Added

* everything! first commit.
