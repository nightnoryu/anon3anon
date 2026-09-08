# Configuration

All configuration is supplied through environment variables. Every variable is
prefixed with `ANON3ANON_`. There is no configuration file.

## Reference

| Variable                             | Required | Default              | Description                                                                                                                                                                                                                                                  |
|--------------------------------------|----------|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ANON3ANON_TELEGRAM_BOT_TOKEN`       | yes      | -                    | Bot token from [@BotFather](https://t.me/BotFather). The process exits on startup if it is missing.                                                                                                                                                          |
| `ANON3ANON_PSEUDONYM_KEY`            | yes      | -                    | Master key used to pseudonymize sender identifiers at rest. Base64 (standard or URL alphabet, padded or not) or hex, decoding to **at least 32 bytes**. Generate with `openssl rand -base64 32`. The process exits on startup if it is missing or too short. |
| `ANON3ANON_DATABASE_PATH`            | no       | `/data/anon3anon.db` | Path to the SQLite database file. The parent directory must be writable by the process user.                                                                                                                                                                 |
| `ANON3ANON_ALLOWED_USER_IDS`         | no       | *(empty = everyone)* | Comma-separated Telegram user IDs permitted to register as recipients. Empty means anyone may register. Senders are never restricted by this list. Look up an ID with [@userinfobot](https://t.me/userinfobot).                                              |
| `ANON3ANON_RATE_LIMIT_WINDOW`        | no       | `1h`                 | Rate-limit bucket size, as a Go duration (`30m`, `2h`, `24h`). `0` disables rate limiting.                                                                                                                                                                   |
| `ANON3ANON_RATE_LIMIT_MAX`           | no       | `100`                | Maximum inbound anonymous messages per sender -> recipient pair per window. Recipient replies are not counted. `0` disables rate limiting.                                                                                                                   |
| `ANON3ANON_HEALTH_ADDR`              | no       | `:8080`              | Listen address for the health HTTP server that serves `/healthz` and `/readyz`.                                                                                                                                                                              |
| `ANON3ANON_RETENTION_AGE`            | no       | `720h`               | Idle age, as a Go duration, after which a background sweep deletes `sessions` (the timer is bumped by each inbound message), `relays`, `blocks`, and `message_rates` rows. `0` disables retention pruning.                                                   |
| `ANON3ANON_RETENTION_SWEEP_INTERVAL` | no       | `1h`                 | How often the retention sweep runs, as a Go duration. `0` disables it.                                                                                                                                                                                       |
| `ANON3ANON_LOG_LEVEL`                | no       | `info`               | One of `debug`, `info`, `warn`, `error`.                                                                                                                                                                                                                     |

## Notes

### `ANON3ANON_PSEUDONYM_KEY`

This key is what stops the database from naming its own users. The stored
`sessions`, `relays`, `blocks`, and `message_rates` rows hold only a keyed
reference derived from a sender's chat ID, never the ID itself, so a stolen
database file, volume snapshot, or backup cannot link a conversation to a
Telegram account without the key.

- **Keep it off the data volume.** Store it wherever your other secrets live, not
  next to the database file it protects.
- **Keep it stable per database.** Losing or changing the key does not corrupt
  anything, but every session, block, and relay written under the old key stops
  matching: senders have to reopen their link and blocks have to be reissued.
- Hex is decoded before base64, because the two encodings overlap for some
  strings. Prefer the `openssl rand -base64 32` output as-is.

### `ANON3ANON_LOG_LEVEL`

At `info` and above, nothing that identifies a sender or repeats what they wrote
is written to stdout. **`debug` logs message text, usernames, and raw user IDs.**
Use `debug` only while diagnosing a problem, and lower it again afterward.

### `ANON3ANON_ALLOWED_USER_IDS`

The allow list restricts registration as a *recipient* only. Anyone who opens a
recipient's personal link can send anonymous messages regardless of this setting.
Leave it empty to let anyone register.

### Rate limiting

Rate limiting is per sender -> recipient pair. One recipient being spammed does not
consume any other recipient's quota. Set either `ANON3ANON_RATE_LIMIT_WINDOW` or
`ANON3ANON_RATE_LIMIT_MAX` to `0` to disable it entirely. Failed deliveries do not
consume quota.

### Durations

Duration values are parsed by Go's `time.ParseDuration`: `300ms`, `1h30m`, `24h`.
There is no unit larger than `h`, so a 30-day retention age is `720h`.
