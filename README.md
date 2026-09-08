# anon3anon

<a href="https://github.com/nightnoryu/anon3anon/releases"><img src="https://img.shields.io/github/release/nightnoryu/anon3anon.svg?cache-control=no-cache"></a>
<a href="https://github.com/nightnoryu/anon3anon/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/anon3anon?cache-control=no-cache"></a>
<a href="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>

Multi-tenant Telegram bot for anonymous messages. One bot instance serves many recipients at once - every registered
user gets their own personal link, and anyone who opens that link can message them anonymously. Public
instance: [@anon3anon_bot](https://t.me/anon3anon_bot).

## ✅ Features

- **Per-user personal links** - `t.me/<bot>?start=<token>`, with unguessable random tokens
- **Two-way threaded conversations** - the recipient replies to a delivered message and the answer goes back to the
  original anonymous sender, still anonymous in both directions
- **`/revoke`** rotates your link, kills the old one, and drops every routing session opened through it, so senders
  who already had the link can no longer reach you
- **`/block`** as a reply to a delivered message stops that one anonymous sender from ever reaching you again - other
  senders are unaffected
- **Per-pair rate limiting** - each sender is capped at *N* messages per time window *per recipient*, so one recipient
  getting spammed doesn't affect anyone else
- **Optional allow list** - restrict who may register as a recipient by Telegram user ID; senders are never restricted
- Messages are relayed with `copyMessage`, so any content type (text, photos, files, voice, stickers, etc.) works and
  no "forwarded from" header leaks the sender - except shared contacts and locations/venues, which are rejected in
  both directions because they carry PII
- **`/delete`** erases your account and every row tied to it - link, sessions, relay history, blocks, rate counters -
  in one irreversible step
- **Senders are pseudonymized at rest** - the database never stores an anonymous sender's Telegram ID, only a keyed
  reference to it, so a stolen database file, volume snapshot, or backup cannot name who wrote to whom without the
  key, which lives in the deployment's secret store
- **Message content stays out of the logs** - at the default log level nothing that identifies a sender or repeats
  what they wrote is written to stdout

## 💬 How it works

**As a recipient**

1. Send `/start` to the bot. It replies with your personal link
2. Share that link with anyone you want anonymous messages from
3. Their messages arrive in your chat with the bot. **Reply** to a message to answer its sender
4. `/mylink` shows the link again, `/revoke` issues a fresh one, `/block` (as a reply) bans a sender
5. `/delete` erases your account and all associated data - irreversible
6. `/help` explains every command

**As a sender**

1. Open someone's personal link (`t.me/<bot>?start=<token>`). The bot confirms you can now write
2. Send messages normally - they are delivered anonymously to the link's owner
3. When the owner replies, their answer lands in your chat. Reply to it to continue the thread
4. `/stop` leaves the conversation - your messages go nowhere until you open a link again

## 🚀 Hosting

You can easily host your own instance of this bot.

### Docker

Just run the pre-built image:

```shell
docker run -d --name anon3anon \
  -e ANON3ANON_TELEGRAM_BOT_TOKEN=123:ABC \
  -e ANON3ANON_PSEUDONYM_KEY="$(openssl rand -base64 32)" \
  -e ANON3ANON_ALLOWED_USER_IDS=123,456 \
  -v anon3anon-data:/data \
  ghcr.io/nightnoryu/anon3anon:latest
```

> **Keep `ANON3ANON_PSEUDONYM_KEY` safe and stable.** It is what stops the database from naming its own users, so
> store it wherever your other secrets live - never on the data volume next to the database. Losing it or changing it
> does not corrupt anything, but every session, block, and relay written under the old key stops matching: senders
> have to reopen their link and blocks have to be reissued.

Or with docker-compose:

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

volumes:
  anon3anon-data:
```

### Kubernetes

`k8s/` holds a Kustomize setup (`base` + `prod` overlay):

- Single replica, `Recreate` strategy, SQLite on a `PersistentVolumeClaim`
- Secrets are SOPS-encrypted (age) and decrypted at apply time
  with [ksops](https://github.com/viaduct-ai/kustomize-sops)
- A [Cloudflare WARP](https://github.com/cmj2002/warp-docker) init container gives the app a SOCKS5 proxy for Telegram
  egress where the API is blocked

The secret must carry `ANON3ANON_TELEGRAM_BOT_TOKEN` and `ANON3ANON_PSEUDONYM_KEY`; the deployment refuses to start
without either. Replace the sops-encoded secrets with yours and apply the `prod` overlay:

```shell
kustomize build --enable-alpha-plugins --enable-exec k8s/prod | kubectl apply -f -
```

### Configuration

All configuration is set via environment variables (prefix `ANON3ANON_`):

| Variable                             | Required | Default              | Description                                                                                                                                                                             |
|--------------------------------------|----------|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ANON3ANON_TELEGRAM_BOT_TOKEN`       | yes      | —                    | Bot token from [@BotFather](https://t.me/BotFather)                                                                                                                                     |
| `ANON3ANON_PSEUDONYM_KEY`            | yes      | —                    | Key used to pseudonymize sender identifiers at rest, as base64 or hex, at least 32 bytes. Generate with `openssl rand -base64 32`, keep it off the data volume, and keep it stable      |
| `ANON3ANON_DATABASE_PATH`            | no       | `/data/anon3anon.db` | Path to the SQLite database file                                                                                                                                                        |
| `ANON3ANON_ALLOWED_USER_IDS`         | no       | *(empty = everyone)* | Comma-separated Telegram user IDs permitted to register as recipients. Can be obtained from [@userinfobot](https://t.me/userinfobot)                                                    |
| `ANON3ANON_RATE_LIMIT_WINDOW`        | no       | `1h`                 | Rate-limit bucket size (Go duration). `0` disables rate limiting                                                                                                                        |
| `ANON3ANON_RATE_LIMIT_MAX`           | no       | `100`                | Max inbound anonymous messages per sender -> recipient per window. Recipient replies are not counted. `0` disables rate limiting                                                        |
| `ANON3ANON_HEALTH_ADDR`              | no       | `:8080`              | Listen address for the liveness (`/healthz`) and readiness (`/readyz`) HTTP endpoints                                                                                                   |
| `ANON3ANON_RETENTION_AGE`            | no       | `720h`               | Idle age (Go duration) after which a background sweep deletes `sessions` (bumped by each inbound message), `relays`, `blocks`, and `message_rates` rows. `0` disables retention pruning |
| `ANON3ANON_RETENTION_SWEEP_INTERVAL` | no       | `1h`                 | How often the retention sweep runs (Go duration). `0` disables it                                                                                                                       |
| `ANON3ANON_LOG_LEVEL`                | no       | `info`               | `debug`, `info`, `warn` or `error`. **`debug` logs message text, usernames, and raw user IDs** - use it only while diagnosing a problem                                                 |

## ⚒️ Local Development

### Prerequisites

- [mise](https://mise.jdx.dev)
- Docker with docker-compose-plugin

### First launch

```shell
git clone https://github.com/nightnoryu/anon3anon
cd anon3anon

# Configure the environment (bot token)
cp compose.override.example.yml compose.override.yml
$EDITOR compose.override.yml

mise run        # download modules, build the binary, lint, test
mise run dev    # build and start the container (binary is bind-mounted from ./bin)
```

The container runs the host-built binary from `./bin`, so after a code change:

```shell
mise run dev:reload   # rebuild and restart the container
```

See [mise.toml](/mise.toml) for more pre-configured tasks.

## 📜 License

Distributed under the MIT License. See [License](/LICENSE) for more information.
