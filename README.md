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

## Status

Work in progress / learning project. Next steps: configurable check interval, structured logging, and proper handling of `git pull` failures (e.g. diverged branches, merge conflicts).
