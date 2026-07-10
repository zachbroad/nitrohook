# Changelog

All notable changes to NitroHook are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-10

### Added

- `make dev` — one-command local development loop: starts Postgres and Redis in
  Docker (waiting until healthy), applies migrations via the API binary, then runs
  the API with an in-process worker under [Air](https://github.com/air-verse/air)
  for hot reload.
- `make dev-setup` — installs the Air hot-reload tool.
- `make dev-down` — stops the Postgres and Redis containers started by `make dev`.

[Unreleased]: https://github.com/zachbroad/nitrohook/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/zachbroad/nitrohook/releases/tag/v0.1.0
