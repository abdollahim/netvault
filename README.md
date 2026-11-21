# netvault

Scheduled SSH collection and change tracking for network devices.

> **Status: in development.** The core connection and shell driver work
> against real hardware. Scheduling, archive storage, git history, and the
> service are not built yet. See [CHANGELOG.md](CHANGELOG.md).

## What it does

netvault connects to network devices over SSH, runs the read-only commands
you define, and stores each output. It is designed to:

- run collection on a weekly schedule, by brand, group, or all devices
- store each output as a timestamped file, per device and per run
- track configuration changes per device in git
- retry failed devices within a run and carry unfinished ones to the next scheduled run

Items in the last two bullets are planned and not yet implemented.

## Current capabilities

- SSH with legacy key exchange, cipher, and MAC algorithms, enabled explicitly
  for older devices
- Host key pinning: the first key is trusted and stored, and a changed key
  stops the connection
- Interactive shell sessions with prompt detection, pager handling, and enable mode
- Brand profiles: prompts, paging, enable command, timeouts, and SSH algorithms
  are defined in a profile file, so adding a brand does not require code changes
- Completeness check for running configurations

## Safety

- The device account must be **read-only**. netvault runs only the commands
  listed in your tasks file, and it never sends configuration changes.
- Archived output and logs can contain sensitive data, such as password hashes
  and SNMP strings. Keep the archive and logs readable only by the service user.
- Real inventory, task, credential, and host key pin files stay out of the
  repository. They are listed in `.gitignore`, and examples are in `examples/`.

## Requirements

- Go 1.23 or newer
- Linux with systemd for deployment
- Development and tests run on any platform with Go

## Setup for development

```bash
go vet ./...
go test ./...
```

Build a Linux binary from any platform:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/netvault ./cmd/netvault
```

## Configuration

Copy the files in `examples/` to `config/` and edit them. Files in `config/`
are ignored by git. Local runtime data, such as host key pins, goes in
`local/`, which is also ignored.

| File | Purpose |
|---|---|
| `config/config.yaml` | Paths, logging, and schedules |
| `config/devices.yaml` | Device names, addresses, brands, and groups |
| `config/tasks.yaml` | Command lists and their archive or raw groups |
| `profiles/*.yaml` | Brand definitions, committed to the repository |

## License

No license has been chosen yet. Until one is added, the code is not licensed
for reuse.