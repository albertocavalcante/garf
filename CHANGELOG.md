# Changelog

## [0.5.4](https://github.com/albertocavalcante/garf/compare/v0.5.3...v0.5.4) (2025-06-10)


### Features

* **jfrog_url:** normalize JFrog URL to ensure presence of `/artifactory` if missing ([#126](https://github.com/albertocavalcante/garf/issues/126)) ([dc087e3](https://github.com/albertocavalcante/garf/commit/dc087e37eb6c81d663c5ef79918f59ba1a7277f3))


### Bug Fixes

* **deps:** update module golang.org/x/tools to v0.34.0 ([#123](https://github.com/albertocavalcante/garf/issues/123)) ([22051e5](https://github.com/albertocavalcante/garf/commit/22051e561550b1a12091b78052614247ce04fef2))

## [0.5.3](https://github.com/albertocavalcante/garf/compare/v0.5.2...v0.5.3) (2025-06-01)


### Bug Fixes

* **mirror:** improve logging and catch bugs for preserving url paths when the source strip content is empty ([#117](https://github.com/albertocavalcante/garf/issues/117)) ([9940673](https://github.com/albertocavalcante/garf/commit/99406733c510087f1feb6cc69ce8ebaa054b6d69))

## [0.5.2](https://github.com/albertocavalcante/garf/compare/v0.5.1...v0.5.2) (2025-05-28)


### Bug Fixes

* source path stripping preserves GitHub structure ([#113](https://github.com/albertocavalcante/garf/issues/113)) ([4cbd5f4](https://github.com/albertocavalcante/garf/commit/4cbd5f4c822fab86c234d774c8c4a7f90b635dc9))

## [0.5.1](https://github.com/albertocavalcante/garf/compare/v0.5.0...v0.5.1) (2025-05-28)


### Bug Fixes

* strip prefix behavior to keep the rest of the url prefix ([#110](https://github.com/albertocavalcante/garf/issues/110)) ([c3bf43e](https://github.com/albertocavalcante/garf/commit/c3bf43eed76a86cfe8cfecc8b3e36bfa61941299))

## [0.5.0](https://github.com/albertocavalcante/garf/compare/v0.4.1...v0.5.0) (2025-05-28)


### ⚠ BREAKING CHANGES

* add `generic` source type and handle it better ([#108](https://github.com/albertocavalcante/garf/issues/108))

### Features

* add `generic` source type and handle it better ([#108](https://github.com/albertocavalcante/garf/issues/108)) ([4b0c245](https://github.com/albertocavalcante/garf/commit/4b0c245d99a506db56f5fbcac5b218c1a47e5cb4))

## [0.4.1](https://github.com/albertocavalcante/garf/compare/v0.4.0...v0.4.1) (2025-05-28)


### Bug Fixes

* expose `source-path-strip` flag in the cli ([#106](https://github.com/albertocavalcante/garf/issues/106)) ([7be6ac3](https://github.com/albertocavalcante/garf/commit/7be6ac3d0ac99bda568f0c35c6b0620f92b77979))

## [0.4.0](https://github.com/albertocavalcante/garf/compare/v0.3.4...v0.4.0) (2025-05-28)


### ⚠ BREAKING CHANGES

* fix race condition and add support for strip prefix ([#104](https://github.com/albertocavalcante/garf/issues/104))

### Features

* fix race condition and add support for strip prefix ([#104](https://github.com/albertocavalcante/garf/issues/104)) ([9655fd3](https://github.com/albertocavalcante/garf/commit/9655fd38aa01a16d7252098c7535a1b787cb9444))

## [0.3.4](https://github.com/albertocavalcante/garf/compare/v0.3.3...v0.3.4) (2025-05-27)


### Bug Fixes

* `preserve-zip-name` flag behavior ([#101](https://github.com/albertocavalcante/garf/issues/101)) ([f50e0cd](https://github.com/albertocavalcante/garf/commit/f50e0cd0fa8e2367826a72f4e897805eeb8459cc))

## [0.3.3](https://github.com/albertocavalcante/garf/compare/v0.3.2...v0.3.3) (2025-05-27)


### Features

* add flag `preserve-zip-name` to keep zip name for the file contents ([#99](https://github.com/albertocavalcante/garf/issues/99)) ([c76a702](https://github.com/albertocavalcante/garf/commit/c76a7025930fe4eec8634325e8ac858373e83b0d))

## [0.3.2](https://github.com/albertocavalcante/garf/compare/v0.3.1...v0.3.2) (2025-05-27)


### Bug Fixes

* zip extraction during mirror phase ([#97](https://github.com/albertocavalcante/garf/issues/97)) ([a9797d9](https://github.com/albertocavalcante/garf/commit/a9797d999b75a88fd4451ea9cadc49a3bfc26083))

## [0.3.1](https://github.com/albertocavalcante/garf/compare/v0.3.0...v0.3.1) (2025-05-27)


### Features

* enhance logging ([#95](https://github.com/albertocavalcante/garf/issues/95)) ([181992b](https://github.com/albertocavalcante/garf/commit/181992b51c02954ed2bc3d9e67354032870e7644))

## [0.3.0](https://github.com/albertocavalcante/garf/compare/v0.2.0...v0.3.0) (2025-05-27)


### ⚠ BREAKING CHANGES

* Expose Library API for 3rd Party Usages ([#93](https://github.com/albertocavalcante/garf/issues/93))

### Features

* Expose Library API for 3rd Party Usages ([#93](https://github.com/albertocavalcante/garf/issues/93)) ([44afc3e](https://github.com/albertocavalcante/garf/commit/44afc3e971d1a47221b65750d506a35f4fdfbf1d))

## [0.2.0](https://github.com/albertocavalcante/garf/compare/v0.1.1...v0.2.0) (2025-05-27)


### ⚠ BREAKING CHANGES

* flags behaviors have changed.

### Features

* add support for `.netrc` ([#86](https://github.com/albertocavalcante/garf/issues/86)) ([e8fc90e](https://github.com/albertocavalcante/garf/commit/e8fc90e7a5af85e70fe75c8e9e034e181a717386))
* Reorganize Package Structure for Better Modularity ([#73](https://github.com/albertocavalcante/garf/issues/73)) ([34740fe](https://github.com/albertocavalcante/garf/commit/34740fe0380128556b04a6c7816425f9e0792c33))
* revamp `garf CLI` ([#68](https://github.com/albertocavalcante/garf/issues/68)) ([06035ce](https://github.com/albertocavalcante/garf/commit/06035ced8b93120100cb2934e869accd7789c2cc))


### Bug Fixes

* **deps:** update module github.com/jfrog/jfrog-client-go to v1.52.0 ([#79](https://github.com/albertocavalcante/garf/issues/79)) ([58ef25b](https://github.com/albertocavalcante/garf/commit/58ef25ba8f0486273ad2bb784472bab45f96e0c1))
* **deps:** update module golang.org/x/tools to v0.32.0 ([#77](https://github.com/albertocavalcante/garf/issues/77)) ([4b23525](https://github.com/albertocavalcante/garf/commit/4b23525dc110a6765a9a1a3a3a7cc1ac4ee2a33e))
* **deps:** update module golang.org/x/tools to v0.33.0 ([#87](https://github.com/albertocavalcante/garf/issues/87)) ([90e12a6](https://github.com/albertocavalcante/garf/commit/90e12a64a7157ce5dc19750545489d6f3dff1e67))
* handle destination and parsing rules properly ([#75](https://github.com/albertocavalcante/garf/issues/75)) ([6c43241](https://github.com/albertocavalcante/garf/commit/6c4324153b6b15b190937852b6c669cc1489da60))
* version test to properly support parallel testing ([#76](https://github.com/albertocavalcante/garf/issues/76)) ([a678754](https://github.com/albertocavalcante/garf/commit/a6787547dc839f8f7b124f9aa63770d3f8a9850b))

## [0.1.1](https://github.com/albertocavalcante/garf/compare/v0.1.0...v0.1.1) (2025-03-28)


### Features

* add `version` command ([#66](https://github.com/albertocavalcante/garf/issues/66)) ([78dec1e](https://github.com/albertocavalcante/garf/commit/78dec1ee372c338f6b3617c3c960081d75c26fb6))

## 0.1.0 (2025-03-27)


### Features

* add `--properties` flag to include custom metadata ([#43](https://github.com/albertocavalcante/garf/issues/43)) ([04e027c](https://github.com/albertocavalcante/garf/commit/04e027cc2f68ff3b1a48c9752adf5a8cb6469173))
* add `--unzip` flag ([#46](https://github.com/albertocavalcante/garf/issues/46)) ([4f76844](https://github.com/albertocavalcante/garf/commit/4f768444612fda5c621064339ef77c14e2dc2c99))


### Bug Fixes

* version path missing from destination artifact URL path ([9442a02](https://github.com/albertocavalcante/garf/commit/9442a02bd76a44e84ed89b6e1809c4fab1024752))

## 0.1.0 (Unreleased)

Initial release
