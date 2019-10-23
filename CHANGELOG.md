# Changelog
All notable changes to this project will be documented in this file.  The
format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

This project tracks the **major** and **minor** versions set by
[`h3`](github.com/uber/h3), and introduces backwards-compatible updates and/or
fixes via patches with patch version bumps.

## Unreleased

### Added

* Some useful H3 constants (#22):
  * `MaxResolution`
  * `NumIcosaFaces`
  * `NumBaseCells`

## 3.0.1

### Added

* Polyfill function (#19).

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
