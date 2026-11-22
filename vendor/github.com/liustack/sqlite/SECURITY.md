# Security Policy

## Reporting a vulnerability

Please report security issues privately, not in a public issue.

Use [GitHub Security Advisories](https://github.com/liustack/sqlite/security/advisories/new) to open a private report. That keeps the details between you and the maintainer until a fix is out.

Include what a bug report needs: the smallest program or SQL that shows it, the full output, the module version, `go version`, and the OS and architecture. We will acknowledge the report, work on a fix, and credit you unless you prefer otherwise.

## Supported versions

Fixes land in the latest release (the newest `v0.1.x` tag of github.com/liustack/sqlite). Upgrade to it before reporting.

## What to keep in mind

The engine is SQLite as released, unchanged: [docs/upstream.md](docs/upstream.md) says which release it is. A flaw in SQLite itself is fixed by SQLite, and the driver moves to that release. Report it here when this driver ships an affected release, or when the flaw is in what the driver adds: the Go host, the VFS, group commit, or the replicator, which signs requests to S3 with your credentials.
