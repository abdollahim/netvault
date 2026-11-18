# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com),
and the project uses [Semantic Versioning](https://semver.org).

## [Unreleased]

## [0.0.1] - 2026-11-18

Pre-release. The project is not yet usable: it has no device connections and runs no commands.

### Added
- Project scaffold with layered package layout: domain, config, transport, driver, store, scheduler, service, cli.
- Example configuration, device, and task files with placeholder values only.
- Test that rejects site-specific data, such as private addresses and system paths, in committed examples.
- `.gitattributes` forcing LF line endings in the repository.

[Unreleased]: https://github.com/abdollahim/netvault/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/abdollahim/netvault/releases/tag/v0.0.1