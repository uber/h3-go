# Changelog
All notable changes to this project will be documented in this file.  The
format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

This project tracks the **major** and **minor** versions set by
[`h3`](github.com/uber/h3), and introduces backwards-compatible updates and/or
fixes via patches with patch version bumps.

## Unreleased

### Added

* Polyfill function

### Changed

* [breaking] `Uncompat` now returns `([]H3Index, error)` instead of `[]H3Index` 
  to accommodate error scenario from C API.

### Fixed

* panic when using `Uncompact` with invalid resolutions.

## v3.0.0

### Added

* everything! first commit.
