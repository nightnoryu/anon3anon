# Repository Guidelines

## Overview

anon3anon is an anonymous contact relay for Telegram. It uses long-polling to receive
updates from Telegram and pseudonymises identities using HMAC with AES-256 seals.

## Project Structure & Module Organization

`cmd/anon3anon/` contains process wiring: configuration, logging, the health
server, retention worker, and `main`. Keep business types and the storage port
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
mise run              # download modules, build, lint, and run all unit tests

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
Wrap errors with useful operation context (for example, `fmt.Errorf("open db: %w", err)`).

## Testing Guidelines

Place tests next to the code as `*_test.go`. External-facing package tests use
the `package_name_test` convention and `testify` assertions. Name tests after
observable behavior, such as `TestNew` or `TestRevokeRemovesRelays`; call
`t.Parallel()` when isolation permits. Add regression coverage for routing,
privacy, persistence, or command behavior changes, then run `mise run` before
opening a PR.

## Definition of Done

A task is considered done only when all of the following pass without errors:

1. Full build & unit tests - run:

```shell
mise run
```

Do not mark work complete if any build error, lint warning, unit test failure, or
integration test failure remain unresolved.

## Commit & Pull Request Guidelines

Use concise imperative commit subjects, capitalized and without a trailing
period: `Add user IDs scope`, `Fix linter warning`, or `Adjust healthcheck for warp`.
Keep each commit focused. PRs should explain the behavior change and rationale,
link the relevant issue when one exists, note configuration or deployment
changes, and include screenshots only for user-visible interface changes. Confirm
that `mise run` passes and update `README.md` or `docs/` when operations or
configuration change.
