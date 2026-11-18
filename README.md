# netvault

Scheduled SSH collection and change tracking for network devices.

> Status: in development. Not yet usable. See CHANGELOG.md.

## What it does

- Connects to devices on a weekly schedule, grouped by brand and site.
- Runs the read-only commands you define and stores each output as a timestamped file.
- Tracks configuration changes per device in git.
- Retries failed devices within a run and carries unfinished ones to the next scheduled run.

## Safety

- The device account must be read-only. netvault runs only the commands listed in your tasks file.
- Archived output and logs can contain sensitive data. Keep the archive and logs readable only by the service user.
- Real inventory, task, and credential files stay out of the repository. Examples are in `examples/`.

## Requirements

- Linux with systemd
- Go 1.23 or newer

## Configuration

Copy the files in `examples/` to `config/` and edit them. Files in `config/` are ignored by git.

## License

See LICENSE.