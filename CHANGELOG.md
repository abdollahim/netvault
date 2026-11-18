# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com),
and the project uses [Semantic Versioning](https://semver.org).

## [Unreleased]

## [0.1.0] - 2026-11-18

Pre-release. The first real connection to a network device works.

### Added
- Legacy SSH transport for devices that offer only older key exchange, cipher, and MAC algorithms.
- Host key pinning: the first key is trusted and stored, and a later change stops the connection.
- Throwaway spike command that runs one command on a device and reports output size and time. It is removed in a later phase.
- Tests against an in-process legacy device covering command execution, refused modern-only clients, wrong passwords, and changed host keys.

### Measured
- `show version` on a Cisco REDACTED-MODEL series switch: about 2.4 KB, about 0.7 s, with no algorithm fallback.

[Unreleased]: https://github.com/abdollahim/netvault/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/abdollahim/netvault/releases/tag/v0.1.0
[0.0.1]: https://github.com/abdollahim/netvault/releases/tag/v0.0.1