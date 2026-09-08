# Local Development

This project builds a static Linux binary on the host and runs it inside a
container that bind-mounts `./bin`. mise drives every task, Docker runs the container.

## Prerequisites

- [mise](https://mise.jdx.dev) - provisions the Go and `golangci-lint` versions
  pinned in [`mise.toml`](../mise.toml) and runs the task scripts
- Docker with the `docker-compose-plugin`

## First launch

```shell
git clone https://github.com/nightnoryu/anon3anon
cd anon3anon

# Local secrets live in an untracked compose override
cp compose.override.example.yml compose.override.yml
$EDITOR compose.override.yml   # set ANON3ANON_TELEGRAM_BOT_TOKEN and ANON3ANON_PSEUDONYM_KEY

mise run        # download modules, build ./bin/anon3anon, lint, run unit tests
mise run dev    # build, then start the container and wait for it to be healthy
```

`mise run` with no task name runs the `default` task: `build` followed by `check` (`lint` + `test:unit`).

### The compose override

[`compose.yml`](../compose.yml) holds the non-secret defaults and mounts
`./bin:/app/bin:ro` so the container executes the host-built binary.
`compose.override.yml` is git-ignored and supplies the two secret values:

```yaml
services:
  anon3anon:
    environment:
      ANON3ANON_TELEGRAM_BOT_TOKEN: "123:ABC"
      ANON3ANON_PSEUDONYM_KEY: "<output of: openssl rand -base64 32>"
```

Generate the pseudonym key once and keep it stable, otherwise data written under
the old key stops matching. See [configuration.md](configuration.md) for the full
variable list.

## The edit / rebuild loop

The container runs `./bin/anon3anon` from the bind mount, so a code change is not
picked up until the binary is rebuilt and the container restarts:

```shell
mise run dev:reload   # rebuild ./bin, then `docker compose restart anon3anon`
```

## Tasks

Defined in [`mise.toml`](../mise.toml):

| Task                  | Action                                                                         |
|-----------------------|--------------------------------------------------------------------------------|
| `mise run`            | `build` then `check` (the `default` task)                                      |
| `mise run build`      | `go build -trimpath -o ./bin/anon3anon ./cmd/anon3anon` (depends on `modules`) |
| `mise run modules`    | `go mod download`                                                              |
| `mise run tidy`       | `go mod tidy`                                                                  |
| `mise run check`      | `lint` + `test:unit`                                                           |
| `mise run lint`       | `golangci-lint run`                                                            |
| `mise run test:unit`  | `go test ./...`                                                                |
| `mise run dev`        | build, then `docker compose up -d --wait`                                      |
| `mise run dev:reload` | build, then `docker compose restart anon3anon`                                 |
| `mise run dev:down`   | `docker compose down`                                                          |
| `mise run dev:ps`     | `docker compose ps`                                                            |
| `mise run dev:logs`   | `docker compose logs -f`                                                       |

The build environment is fixed by `mise.toml`: `GOOS=linux`, `GOARCH=amd64`,
`CGO_ENABLED=0` (the SQLite driver is `modernc.org/sqlite`, pure Go - no C
toolchain needed).
