# Self-Hosted Deployment

anon3anon ships as a single static binary in a minimal Alpine image published to
`ghcr.io/nightnoryu/anon3anon`. It needs one writable volume for the SQLite
database and outbound access to the Telegram Bot API. It uses long polling, so it
needs **no inbound ports** other than the optional health endpoint.

For the full list of environment variables see [configuration.md](configuration.md).

## Before you start

- A bot token from [@BotFather](https://t.me/BotFather).
- A pseudonym key: `openssl rand -base64 32`. Store it with your other secrets,
  **not** on the data volume, and keep it stable - see [configuration.md](configuration.md#anon3anon_pseudonym_key).
- Somewhere for `/data` to persist across restarts.

## Docker

```shell
docker run -d --name anon3anon \
  -e ANON3ANON_TELEGRAM_BOT_TOKEN=123:ABC \
  -e ANON3ANON_PSEUDONYM_KEY="$(openssl rand -base64 32)" \
  -e ANON3ANON_ALLOWED_USER_IDS=123,456 \
  -v anon3anon-data:/data \
  ghcr.io/nightnoryu/anon3anon:latest
```

Pin a released tag (for example `:1.0.0`) instead of `:latest` for reproducible
deployments.

## Docker Compose

```yaml
services:
  anon3anon:
    image: ghcr.io/nightnoryu/anon3anon:latest
    container_name: anon3anon
    restart: unless-stopped
    environment:
      ANON3ANON_TELEGRAM_BOT_TOKEN: "123:ABC"
      ANON3ANON_PSEUDONYM_KEY: "<output of: openssl rand -base64 32>"
      ANON3ANON_ALLOWED_USER_IDS: "123,456"
    volumes:
      - "anon3anon-data:/data"
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://localhost:8080/readyz"]
      interval: 20s
      timeout: 3s
      retries: 3

volumes:
  anon3anon-data:
```

## Health endpoints

The container serves an HTTP health server on `ANON3ANON_HEALTH_ADDR` (default
`:8080`):

| Path       | Meaning                                                                        |
|------------|--------------------------------------------------------------------------------|
| `/healthz` | Liveness. `200 ok` whenever the process is running.                            |
| `/readyz`  | Readiness. Pings the database; `200 ok` on success, `503 not ready` otherwise. |

Wire `/healthz` to a liveness check and `/readyz` to a readiness check. Nothing
else listens on a port.

## Persistence and backups

- The only state is the SQLite database at `ANON3ANON_DATABASE_PATH`
  (default `/data/anon3anon.db`). Keep `/data` on a durable volume.
- The image runs as uid/gid `10001`; the volume must be writable by that user.
  The provided Dockerfile pre-creates `/data` with the right owner.
- To back up, copy the database file (ideally with the container stopped, or via
  SQLite's online backup). A backup is only useful together with the matching
  `ANON3ANON_PSEUDONYM_KEY` - without it the rows cannot be interpreted. Back up
  the key **separately**.

## Upgrades

Single replica only - the database is a local file and there is no cross-process
locking strategy. Stop the old container, start the new image against the same
volume. Schema migrations, if any, run automatically on startup.

## Egress restrictions

Where the Telegram Bot API is blocked at the network level, route the container's
egress through a proxy. The Kubernetes setup below does this with a Cloudflare
WARP sidecar; with plain Docker, set `HTTP_PROXY` / `HTTPS_PROXY` on the
container to your own SOCKS5 or HTTP proxy and add `localhost,127.0.0.1,::1` to
`NO_PROXY`.

## Kubernetes

[`k8s/`](../k8s/) holds a Kustomize configuration: a `base` and a `prod` overlay.

### What the base provides

- `Namespace` `anon3anon`
- `Deployment`: single replica, `Recreate` strategy, non-root
  (`runAsUser`/`runAsGroup` `10001`, `runAsNonRoot`)
- `PersistentVolumeClaim` `anon3anon-data` (`ReadWriteOnce`, 1Gi) mounted at
  `/data`
- A [Cloudflare WARP](https://github.com/cmj2002/warp-docker) init container
  (`restartPolicy: Always`, i.e. a native sidecar) that exposes a SOCKS5 proxy on
  `127.0.0.1:1080`; the app container points `HTTP_PROXY`/`HTTPS_PROXY` at it so
  Telegram egress works where the API is blocked
- Liveness probe on `/healthz`, readiness probe on `/readyz`

### What the prod overlay adds

- A `ConfigMap` (`anon3anon-config`) for the non-secret settings
  (`RATE_LIMIT_*`, `RETENTION_*`, `LOG_LEVEL`) - [`k8s/prod/configmap.yaml`](../k8s/prod/configmap.yaml)
- A SOPS-encrypted `Secret` (`anon3anon-secrets`) decrypted at build time with
  [ksops](https://github.com/viaduct-ai/kustomize-sops) using an
  [age](https://github.com/FiloSottile/age) key - [`k8s/prod/secret.enc.yaml`](../k8s/prod/secret.enc.yaml)
- The image tag pinned via `images:` in [`k8s/prod/kustomization.yaml`](../k8s/prod/kustomization.yaml)

The secret must carry `ANON3ANON_TELEGRAM_BOT_TOKEN` and
`ANON3ANON_PSEUDONYM_KEY`; the deployment refuses to start without either.
`ANON3ANON_ALLOWED_USER_IDS` in the secret is optional.

### Applying the prod overlay

1. Install [SOPS](https://github.com/getsops/sops), [age](https://github.com/FiloSottile/age), and
   [ksops](https://github.com/viaduct-ai/kustomize-sops).
2. Generate your own age key pair. Put the public recipient in [`.sops.yaml`](../.sops.yaml)
   (replacing the existing one).
3. Re-encrypt the secret with your values:

   ```shell
   sops --encrypt --in-place k8s/prod/secret.enc.yaml
   ```

   Edit it with `sops k8s/prod/secret.enc.yaml` to set the plaintext values
   first. Only the `data` / `stringData` fields are encrypted.
4. Adjust the pinned image tag in `k8s/prod/kustomization.yaml` if needed.
5. Make your age private key available to `sops` (for example via
   `SOPS_AGE_KEY_FILE`) and apply:

   ```shell
   kustomize build --enable-alpha-plugins --enable-exec k8s/prod | kubectl apply -f -
   ```

The `--enable-alpha-plugins --enable-exec` flags are required for `ksops` to run
during the build.
