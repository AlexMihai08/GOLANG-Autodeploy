# GOLANG-Autodeploy

A small Go CLI/daemon that watches a directory of staging projects and keeps them in sync with their git remotes and running Docker containers — built as a learning project for Go while solving a real problem with flaky `systemd`-based autodeploy on staging VMs.

## What it does

Given a root path containing one folder per project, on a fixed interval it:

1. Scans all subfolders of the root path.
2. Skips folders that don't contain a `docker-compose.yml`.
3. Records the current `git rev-parse HEAD` for each project.
4. Runs `git pull`.
5. Compares the `HEAD` hash before and after the pull.
6. If the hash changed (new commits were pulled), runs `docker compose up --build -d` for that project.

Each project folder is checked concurrently (one goroutine per folder), and a folder already being processed is skipped on the next tick instead of being run twice in parallel.

## Requirements

- Go 1.26+ (only needed to build the binary)
- `git` and `docker compose` (v2, plugin syntax) available on the target machine
- Each project folder must already be a git repository with a configured remote, and `docker-compose.yml` at its root

## Build

```
go build -o autodeploy .
```

This produces a single static binary — copy it to a staging VM, no Go/PHP/Node runtime needed there.

## Usage

```
./autodeploy -path /var/www/staging
```

- `-path` (required): root directory containing the project folders to watch.

The process runs until stopped (`Ctrl+C`), checking all projects every few seconds.

## Running as a systemd service

The process is a single long-running binary (it already loops internally), so `systemd`'s only job is to keep it running: start it on boot, restart it if it crashes. A ready-to-edit unit file is provided at [`systemd/autodeploy.service`](systemd/autodeploy.service).

### 1. Edit the unit file for your server

Open `systemd/autodeploy.service` and adjust to match your setup:

- `User=` — the Linux user the process should run as (must have access to the project folders, `git`, and `docker`).
- `ExecStart=` — the **absolute path** to the compiled binary, followed by `-path` and your **root folder** (the directory that contains one subfolder per project, each with its own `docker-compose.yml`).

```ini
ExecStart=/home/<user>/GOLANG-Autodeploy/autodeploy -path /path/to/your/projects
```

- `Environment=PATH=...` — leave as-is unless `git`/`docker` are installed somewhere unusual; systemd services don't inherit your shell's `PATH`.

### 2. Install and start it

```bash
sudo cp systemd/autodeploy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now autodeploy
```

- `daemon-reload` — tells systemd to re-read unit files (needed after any change to the `.service` file).
- `enable --now` — enables the service on boot **and** starts it immediately.

### 3. Check status and logs

```bash
sudo systemctl status autodeploy
sudo journalctl -u autodeploy -f
```

- `status` — shows whether it's `active (running)` and the last few log lines.
- `journalctl -u autodeploy -f` — follows the live log output (`Ctrl+C` to stop watching, the service keeps running).

### Common operations

```bash
sudo systemctl restart autodeploy   # apply a new build (after go build -o autodeploy .)
sudo systemctl stop autodeploy      # stop it
sudo systemctl disable autodeploy   # stop starting it on boot
```

## Status

Work in progress / learning project. Next steps: configurable check interval, structured logging, and proper handling of `git pull` failures (e.g. diverged branches, merge conflicts).
