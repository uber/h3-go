# Changelog

This project tracks the **major** and **minor** versions set upstream by
[h3](https://github.com/uber/h3) and introduces backwards-compatible updates and/or
fixes via **patches** with patch version bumps.

## 4.2.4 (6 Jun 2025)

### Added

* [#84]: `GridDisksUnsafe`, `GridDiskDistancesUnsafe`, and `GridDiskDistancesSafe` functions.

### Changed

* [#85]: Convert directly between C and Go arrays.
* [#86]: Slightly optimized `CellsToMultiPolygon` and `LatLng#String`.

Thanks to [@justinhwang] for their contributions to this release.

## 4.2.3 (4 Jun 2025)

### Added

* [#82]: `GridRing` and `GridRingUnsafe` functions.

### Updated

* [#83]: Go was updated to v1.22.

Thanks to [@justinhwang] for their contributions to this release.

## 4.2.2 (31 Mar 2025)

### Fixed

* [#79]: Memory leak in `CellsToMultiPolygon`.

### Updated

* [#79]: H3 Core was updated to v4.2.1.

Thanks to [@zachcoleman] for their contributions to this release.

## 4.2.1 (10 Feb 2025)

### Added

* [#68]: `PolygonToCellsExperimental` function.

Thanks to [@zachcoleman] for their contributions to this release.

## 4.2.0 (27 Dec 2024)

### Breaking Changes

* [#73]: Errors are now returned from various functions.

### Updated

* [#72]: Go was updated to v.1.20.
* [#75]: H3 Core was updated to v4.2.0.

Thanks to [@mojixcoder] for their contributions to this release.

## 4.1.2 (26 Aug 2024)

### Added

* [#71]: Full support for vertices.

Thanks to [@mojixcoder] for their contributions to this release.

## 4.1.1 (12 Aug 2024)

### Added

* [#70]: `CellsToMultiPolygon` function.

Thanks to [@zachcoleman] for their contributions to this release.

## 4.1.0 (22 Mar 2023)

### Added

* [#61]: `CellToChildPos` and `ChildPosToCell` functions.

### Updated

* [#60]: H3 core was updated to v4.1.0.

Thanks to [@akhenakh] for their contributions to this release.

## 4.0.1 (30 Sep 2022)

### Updated

* [c3cc4ae]: H3 core was updated to v4.0.1.

## 4.0.0 (8 Sep 2022)

* [#54]: All new functions to match H3 v4.

See the [migration guide] for upstream changes, and the
[README.md] for upstream to h3-go binding name mapping.

Thanks to [@jogly] for their contributions to this release.

## 3.7.1 (15 Mar 2021)

### Added

* [#46]: Functions to cover full functionality.
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

## 3.7.0 (17 Feb 2021)

### Added

* [#37]: `Line` function.
* [#41]: `SetToLinkedGeo` function.

## 3.0.2 (19 May 2020)

### Fixed

* [#30], [#32]: `go mod vendor` now works correctly.

### Added

* [#22]: Some useful H3 constants:
  * `MaxResolution`
  * `NumIcosaFaces`
  * `NumBaseCells`
* [#24]: Support for GOMODULES.

## 3.0.1 (3 Jun 2019)

### Added

* [#19]: `Polyfill` function.

### Changed

* [#19]: [Breaking] `Uncompat` now returns `([]H3Index, error)` instead of `[]H3Index`
  to accommodate error scenario from C API.

### Fixed

* [#20]: Panic when using `Uncompact` with invalid resolutions.
* [#7], [#9], [#13]: Latitudes and longitudes outside of respective ranges when unprojecting in
  certain areas.

## 3.0.0 (18 Oct 2018)

### Added

* everything! first commit.

[#7]: https://github.com/uber/h3-go/pull/7
[#9]: https://github.com/uber/h3-go/pull/9
[#13]: https://github.com/uber/h3-go/pull/13
[#19]: https://github.com/uber/h3-go/pull/19
[#20]: https://github.com/uber/h3-go/pull/20
[#22]: https://github.com/uber/h3-go/pull/22
[#24]: https://github.com/uber/h3-go/pull/24
[#30]: https://github.com/uber/h3-go/pull/30
[#32]: https://github.com/uber/h3-go/pull/32
[#37]: https://github.com/uber/h3-go/pull/37
[#41]: https://github.com/uber/h3-go/pull/41
[#46]: https://github.com/uber/h3-go/pull/46
[#54]: https://github.com/uber/h3-go/pull/54
[#60]: https://github.com/uber/h3-go/pull/60
[#61]: https://github.com/uber/h3-go/pull/61
[#68]: https://github.com/uber/h3-go/pull/68
[#70]: https://github.com/uber/h3-go/pull/70
[#71]: https://github.com/uber/h3-go/pull/71
[#72]: https://github.com/uber/h3-go/pull/72
[#73]: https://github.com/uber/h3-go/pull/73
[#75]: https://github.com/uber/h3-go/pull/75
[#79]: https://github.com/uber/h3-go/pull/79
[#82]: https://github.com/uber/h3-go/pull/82
[#83]: https://github.com/uber/h3-go/pull/83
[#84]: https://github.com/uber/h3-go/pull/84
[#85]: https://github.com/uber/h3-go/pull/85
[#86]: https://github.com/uber/h3-go/pull/86

[c3cc4ae]: https://github.com/uber/h3-go/commit/c3cc4ae1af0472866452d998fe5576839450e342
[migration guide]: https://h3geo.org/docs/library/migrating-3.x
[README.md]: ./README.md

[@akhenakh]: https://github.com/akhenakh
[@jogly]: https://github.com/jogly
[@justinhwang]: https://github.com/justinhwang
[@mojixcoder]: https://github.com/mojixcoder
[@zachcoleman]: https://github.com/zachcoleman
