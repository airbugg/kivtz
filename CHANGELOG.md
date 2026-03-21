# Changelog

## [0.4.1](https://github.com/airbugg/kivtz/compare/v0.4.0...v0.4.1) (2026-03-21)


### Bug Fixes

* syncignore matches patterns without package prefix ([#10](https://github.com/airbugg/kivtz/issues/10)) ([65ce9d5](https://github.com/airbugg/kivtz/commit/65ce9d5006358ad3b2e164bd4c5d98e59939916f))

## [0.4.0](https://github.com/airbugg/kivtz/compare/v0.3.0...v0.4.0) (2026-03-20)


### Features

* machine-based sync with dry-run ([#9](https://github.com/airbugg/kivtz/issues/9)) ([cfd993f](https://github.com/airbugg/kivtz/commit/cfd993f6fac1c3720029f31ca234f103964ab369))


### Bug Fixes

* chain GoReleaser after release-please in same workflow ([53853b3](https://github.com/airbugg/kivtz/commit/53853b3b8cf3fdf6e12df68045d54d5379f7663f))

## [0.3.0](https://github.com/airbugg/kivtz/compare/v0.2.0...v0.3.0) (2026-03-19)


### Features

* add version.CachedCheck with 24h file cache ([43935ad](https://github.com/airbugg/kivtz/commit/43935ad002ba76f012d24049c8d90ca503d35804))
* add version.CheckForUpdate with GitHub API integration ([fdd302d](https://github.com/airbugg/kivtz/commit/fdd302d85327edd5cffb2565fb3f6bee1bd34f75))
* add version.PrintUpdateNotice with timeout and opt-out ([693eba4](https://github.com/airbugg/kivtz/commit/693eba42515e0b65c167e20fa6ded5d089cfd7fc))
* auto-release pipeline, PR previews, and version check ([1bbe7e5](https://github.com/airbugg/kivtz/commit/1bbe7e5de8b1c886724d0edfe4ad71b2e56a3d9e))
* show update notice after command execution ([db96562](https://github.com/airbugg/kivtz/commit/db96562ada8787c9dc7a9f0a9c378598a0bded45))


### Bug Fixes

* add missing permissions for PR workflows on private repos ([8299724](https://github.com/airbugg/kivtz/commit/82997247ce67c62361bf3935df48e0c05ae38bfb))
* address review feedback for version package ([217ed38](https://github.com/airbugg/kivtz/commit/217ed38ceb48a583852ec930527030c96c667bcb))
* auto-repair dangling symlinks during sync ([0e11b36](https://github.com/airbugg/kivtz/commit/0e11b36ecc8c680229d29e9a93d241128578fc8d))
* use changelog.disable instead of skip for GoReleaser v2 ([1e803e7](https://github.com/airbugg/kivtz/commit/1e803e76568219a2ff3d2a8f0e6656fc9b2e158e))
* use go-version-file in CI and release workflows ([52ad564](https://github.com/airbugg/kivtz/commit/52ad564318c9c8082ea451a8a87b62390e3957a2))
