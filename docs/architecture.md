# Architecture

anon3anon is a single Go process. It long-polls the Telegram Bot API, routes
each incoming message to the other side of an anonymous conversation, and keeps
just enough state in a local SQLite file to make threaded replies work.

```
    sender <-->  Telegram Bot API  <-->  recipient
                        |
                        |  long polling
              +---------------------+
              |      anon3anon      |
              |                     |
              |  middleware         |
              |     |               |
              | handlers -- keyring |   (pseudonymize / encrypt)
              |     |               |
              |  Store              |
              +---------------------+
                        |
                   +---------+
                   | SQLite  |   single connection
                   +---------+
```

Two background goroutines run alongside the update loop: an HTTP health server
(`/healthz`, `/readyz`) and a retention sweeper. Neither is on the message path.

## Layers

```
cmd/anon3anon/                 process wiring: config, logger, keyring, health, retention
pkg/domain/                    core types + the Store port (interface)
pkg/pseudonym/                 keyring: keyed references and authenticated encryption
pkg/token/                     personal-link token generation
pkg/infrastructure/
  telegram/middleware/         private-chat gate, structured logging
  telegram/handler/            command handlers + the message router
  storage/sqlite/              the Store adapter
  health/                      liveness / readiness handlers
```

`pkg/domain` depends on nothing. Everything else depends inward on it. The
handlers talk to storage only through the `domain.Store` interface, so the
SQLite implementation is swappable and is faked in handler tests.

## Request flow

`github.com/go-telegram/bot` delivers each update to a chain:

1. **Private-chat middleware** - drops any update that is not from a private
   chat. Groups and channels are never served.
2. **Logging middleware** - emits one structured line per message. It references
   the chat only by its keyed reference at `info`; raw IDs, usernames, and text
   are `debug`-only.
3. **Handler dispatch** - messages that start with a registered command
   (`/start`, `/help`, `/mylink`, `/revoke`, `/block`, `/stop`, `/delete`) go to
   that command's handler. Everything else goes to the **message router**.

The message router tries, in order:

- **`tryRouteReply`** - if the message is a Telegram reply and
  `(this chat, replied-to message id)` matches a stored relay, copy it to that
  relay's origin and record a relay for the new copy. This is how a threaded
  conversation continues in either direction.
- **`routeToOwner`** - otherwise treat it as a first-contact message: look up the
  sender's session and copy the message to the owner it points at.

Messages are moved with Telegram's `copyMessage`, so any content type works and
no "forwarded from" header leaks the origin. Shared contacts, locations, venues,
stories, and shared user/chat payloads are rejected in both directions because
they carry identity regardless of the text.

## Domain model

### User - a registered recipient

Someone who ran `/start`. Has a Telegram user ID, their chat ID with the bot, an
unguessable `link_token` (the `?start=` value of their personal link), and a
creation time. This is the only entity whose Telegram identity is stored in the
clear - a recipient is not anonymous *to the instance*, only to senders.

`/start <token>` opens a link; `/start` with no payload registers the caller (if
the allow list permits) and returns their link. `/revoke` rotates the token.

### Session - where a sender's next first-contact message goes

A session maps a **sender's chat** to the **owner** they are currently writing
to. It is created when the sender opens a link (`SetSession`) and the last link
opened wins - a sender points at exactly one owner at a time.

A session only routes *first-contact* messages. It is consulted by
`routeToOwner`, bumped (`TouchSession`) on every inbound message so an active
conversation survives retention, and dropped by `/stop` (the sender),
`/revoke` / `/delete` (the owner), or the retention sweep.

### Relay - reply routing for an already-delivered message

Every time the bot copies a message, it records a relay:

```
  key:    (dest_ref, dest_msg_id)      the copy that was delivered
  value:  origin chat, owner, created_at
```

When someone replies to a delivered message, the router looks up
`(their chat, the replied-to message id)`, finds the origin, and copies the
reply there - then writes a relay for *that* new copy so the thread can continue.
Relays are written for both directions, which is why a single `/revoke` that
clears them costs the owner their own ability to answer older threads: inbound
and outbound legs are the same table.

A relay is *not* session-scoped. Clearing a session does not stop replies to
messages already delivered; that needs the relay rows gone too.

## Pseudonymization and encryption

All of this lives in `pkg/pseudonym`. One master key -
`ANON3ANON_PSEUDONYM_KEY`, at least 32 bytes - is loaded at startup. Two subkeys
are derived from it with HKDF-SHA256 under distinct `info` labels:

```
  master key
    |-- HKDF(info=".../chat-ref/v1")  ->  refKey   (HMAC-SHA256 key)
    |-- HKDF(info=".../chat-seal/v1") ->  sealKey  (AES-256 key)
```

Separate labels mean the two uses can never produce overlapping output, and one
subkey leaking does not reveal the other or the master.

### Keyed reference - `Ref(chatID)`

`HMAC-SHA256(refKey, chatID)` truncated to 16 bytes, hex-encoded. Properties the
schema relies on:

- **Deterministic** - the same chat ID always yields the same reference, so it
  works as a primary key (`sessions.sender_ref`) and part of one
  (`relays`, `blocks`, `message_rates`). Dedup, upsert, and "is this sender
  blocked?" are plain key lookups.
- **One-way** - there is no operation that turns a reference back into a chat ID.
- **Keyed** - this is the reason it is HMAC and not a bare `SHA-256(chatID)`.
  Telegram chat IDs are not secret and the space of plausible values is small
  enough to enumerate. A plain hash of a chat ID could be reversed with a
  dictionary in seconds. HMAC under a key that lives in a secret store - not on
  the data volume - makes that offline attack infeasible for someone who has
  only the database file.

The trade-off: because the reference is keyed but not salted per row, the same
sender writing to two different recipients produces the *same* `sender_ref` in
both. Someone with the database alone still cannot name that sender; someone who
also holds the key can confirm a *suspected* chat ID against a stored reference.
See [privacy.md](privacy.md#operator-trust).

### Authenticated encryption - `Seal` / `Open`

A reply eventually has to be delivered to a real chat, so for the origin side of
a relay the actual chat ID must be recoverable - a one-way reference is not
enough. `relays.origin_seal` holds the origin chat ID encrypted with AES-256-GCM
(`sealKey`), a fresh random nonce per write. GCM is authenticated, so a row that
was truncated, tampered with, or written under a different key fails to open
rather than yielding a wrong chat ID. The random nonce means two rows holding the
same origin chat are not recognizable as such from the ciphertext.

`origin_seal` is the only reversible sender identifier in the database, and it
exists only because reply delivery requires it.

## Why SQLite with one connection

The storage adapter opens SQLite through the pure-Go `modernc.org/sqlite` driver
(no cgo, matching the `CGO_ENABLED=0` static build) in WAL mode with
`busy_timeout` and foreign keys on, and then sets:

```go
db.SetMaxOpenConns(1)
```

Reasons:

- **SQLite allows only one writer at a time.** With a pool, concurrent writes
  race for the write lock and lose with `SQLITE_BUSY` / retry noise. Capping the
  pool at one connection turns every database operation into a queue - no
  contention, no busy errors, fully serialized.
- **Several operations are read-modify-write in one statement** (upserts with
  `ON CONFLICT ... RETURNING`, the rate-limit counter bump, token-collision
  retries). Serializing at the connection level keeps their reasoning simple:
  nothing else is writing between the read and the write.
- **The workload is tiny.** A Telegram bot on long polling processes updates
  roughly one at a time; there is no throughput being left on the table. The
  deployment is a single replica on a `ReadWriteOnce` volume with a `Recreate`
  rollout ([deployment.md](deployment.md)), so there is never a second process
  contending for the file either.

The cost is that a slow query blocks the next one. Given the query set - all
point lookups and small deletes on indexed columns - that is not a real
constraint.

## Background work

- **Health server** (`cmd/anon3anon/healthserver.go`) - `/healthz` always returns
  `ok` while the process runs; `/readyz` pings the database and returns `503` if
  that fails. Used by container and Kubernetes probes.
- **Retention sweeper** (`cmd/anon3anon/retention.go`) - every
  `ANON3ANON_RETENTION_SWEEP_INTERVAL` it calls `Store.PurgeExpired`, deleting
  `sessions` / `relays` / `blocks` / `message_rates` rows past
  `ANON3ANON_RETENTION_AGE`. It bounds how long any sender<->recipient linkage is
  retained and keeps those tables from growing without limit. `users` rows are
  never swept. See [privacy.md](privacy.md#retention).

## See also

- [privacy.md](privacy.md) - the threat model these mechanisms serve
- [configuration.md](configuration.md) - every tunable
- [deployment.md](deployment.md) - single-replica deployment and the key/volume split
