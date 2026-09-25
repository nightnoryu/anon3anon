# Repository Guidelines

## Overview

anon3anon is an anonymous contact relay for Telegram. It uses long-polling to receive
updates from Telegram and pseudonymises identities using HMAC with AES-256 seals.

## Project Structure & Module Organization

`cmd/anon3anon/` contains process wiring: configuration, logging, the health
server, metrics, retention worker, and `main`. Keep business types and the storage port
in `pkg/domain/`; this package must not depend on infrastructure. Put Telegram
commands and routing in `pkg/infrastructure/telegram/handler/`, middleware
alongside it, and the SQLite `domain.Store` adapter in
`pkg/infrastructure/storage/sqlite/`. `pkg/pseudonym/` handles keyed references
and encryption; `pkg/token/` generates personal-link tokens. Deployment files
live in `compose.yml`, `Dockerfile`, and `k8s/`; operational documentation is in
`docs/`.

## Build, Test, and Development Commands

Use [mise](https://mise.jdx.dev); `mise.toml` pins Go 1.26 and golangci-lint.

```shell
mise run              # download modules, build, lint, and run unit tests

mise run build        # compile bin/anon3anon
mise run lint         # run golangci-lint, including configured formatters
mise run test:unit    # run go test ./...

mise run dev          # build and start Docker Compose, waiting for health
mise run dev:reload   # rebuild and restart the local service
```

For local Docker use, copy `compose.override.example.yml` to the ignored
`compose.override.yml` and supply the Telegram token and a stable pseudonym key.
Never commit secrets, generated local overrides, or SQLite data.

## Coding Style & Naming Conventions

Write idiomatic Go and let `gofmt`, `goimports`, and `gci` format imports.
Use tabs for indentation, short lowercase package names, exported `PascalCase`
identifiers, and unexported `camelCase` identifiers. Keep dependencies pointing
inward: handlers access persistence through `domain.Store`, not SQLite types.
Wrap errors with useful operation context (for example, `errors.Wrap(err, "open db")`).

## Testing Guidelines

Place tests next to the code as `*_test.go`. Use `package_name_test` when testing
the public API; use the package itself when tests need internal access. Follow
the existing `testify` style. Name tests after observable behavior, and call
`t.Parallel()` when isolation permits. Add focused regression coverage for
meaningful routing, privacy, persistence, or command behavior changes. Avoid
tests that only repeat obvious implementation details.

## Definition of Done

A code change is done only after the full default task passes:

```shell
mise run
```

This runs the build, lint, and unit tests. Resolve any failure before marking a
code change complete. Add comments only where intent or a constraint is not
clear from the code. Keep documentation concise and update it when behavior,
configuration, or operations change; avoid restating obvious code.

## Commit & Pull Request Guidelines

Use concise imperative commit subjects, capitalized and without a trailing
period: `Add user IDs scope`, `Fix linter warning`, or `Adjust healthcheck for warp`.
Keep each commit focused. PRs should explain the behavior change and rationale,
link the relevant issue when one exists, note configuration or deployment
changes, and include screenshots only for user-visible interface changes.
