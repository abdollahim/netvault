# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com),
and the project uses [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added
- Interactive shell session with a PTY, prompt detection, pager handling, and enable mode.
- Completeness check for running configurations: a config is accepted only if it has the `Current configuration` header and ends with `end`.
- Tests for prompt detection, pager removal, and completeness checks, running without a device.
- Spike command requires the device address and the host key pin file as explicit arguments. No site defaults are stored in the code.

### Fixed
- Large running-configuration output was truncated when sent as a single exec command. The device closed the channel after the banner. Commands now run in an interactive shell, the same way a person uses a terminal.

### Measured
- Full running configuration from a production switch through the interactive shell: complete, about 12 seconds.

## [0.1.0] - 2026-11-18

Pre-release. The first real connection to a network device works.

### Added
- Legacy SSH transport for devices that offer only older key exchange, cipher, and MAC algorithms.
- Host key pinning: the first key is trusted and stored, and a later change stops the connection.
- Throwaway spike command that runs one command on a device and reports output size and time. It is removed in a later phase.
- Tests against an in-process legacy device covering command execution, refused modern-only clients, wrong passwords, and changed host keys.

### Measured
- Version command on a production switch: about 0.7 seconds, with no algorithm fallback.

### Known issues
- Large running-configuration output is truncated over exec commands. Fixed in the next release.

[Unreleased]: https://github.com/abdollahim/netvault/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/abdollahim/netvault/releases/tag/v0.1.0
[0.0.1]: https://github.com/abdollahim/netvault/releases/tag/v0.0.1