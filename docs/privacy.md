# Privacy Model

This document describes what anon3anon protects, what it deliberately does not
protect, exactly what a running instance stores, and how retention, deletion, and
revocation remove it.

It describes the code in this repository. A deployment is only as private as its
operator makes it - see [Operator trust](#operator-trust) and
[configuration.md](configuration.md).

## What it protects

- **Identities never cross between the two sides.** A sender and a recipient
  communicate through the bot without either learning the other's Telegram
  account, username, phone number, or display name. Messages are relayed with
  `copyMessage`, so there is no "forwarded from" header.
- **A stolen database does not name anonymous senders.** The database never
  stores an anonymous sender's Telegram user ID or chat ID in the clear. It
  stores a keyed reference derived from the chat ID (see
  [Pseudonymization](#pseudonymization)). Without `ANON3ANON_PSEUDONYM_KEY`,
  which is meant to live in a secret store and not on the data volume, a stolen
  database file, volume snapshot, or backup cannot turn those references back
  into accounts.
- **Message content is never persisted by the bot.** Text, captions, and media
  are passed straight through Telegram's `copyMessage` and are never written to
  the database.
- **Message content stays out of the logs at the default level.** At
  `ANON3ANON_LOG_LEVEL=info` and above, nothing that identifies a sender or
  repeats what they wrote is logged. Structured log lines reference a sender only
  by the same keyed `chat_ref`, never a raw ID or username.
- **PII-bearing message types are rejected in both directions.** Shared contacts,
  locations, venues, stories, and shared-user / shared-chat payloads are refused
  rather than relayed, because they carry identity independent of the text.
- **Abuse is contained per pair.** Rate limiting and `/block` are scoped to a
  single sender -> recipient pair, so one recipient being targeted does not affect
  or reveal anything about other users.

## What it does NOT protect

- **Telegram sees everything.** This is a Bot API application, not MTProto secret
  chats. There is no end-to-end encryption. Telegram (and anyone who can compel
  Telegram) sees every message in full, both parties' real accounts, and the fact
  that they are talking through this bot. anon3anon only stops the *two users*
  from seeing each other.
- **The operator is trusted.** See [Operator trust](#operator-trust) below. Whoever
  runs the instance holds the bot token, the database, the pseudonym key, and the
  logs, and can raise the log level to `debug`. They can de-anonymize senders.
- **Registered recipients are stored in the clear.** A recipient's Telegram user
  ID and chat ID are written to the `users` table verbatim. A stolen database
  names every registered recipient (it just cannot name the anonymous senders
  who wrote to them).
- **Content-level deanonymization.** If a sender types their name, sends a selfie,
  or sends a file whose metadata identifies them, that is relayed verbatim.
  `copyMessage` does not strip EXIF or document metadata. Only the specific
  message *types* listed above are blocked.
- **Traffic and timing analysis.** An adversary who can observe both a sender's
  and a recipient's chat activity can correlate them by timing. The bot adds no
  delay, batching, or cover traffic.
- **A malicious counterparty.** The other side can screenshot, quote, or
  socially engineer. Anonymity is of the transport, not of anything a
  participant chooses to reveal or record.
- **A compromised Telegram account** on either end, or a compromised host running
  the bot.
- **Deletion on Telegram's side.** `/delete` and retention clear the bot's
  database. They cannot remove messages already delivered into the other party's
  Telegram chat history or anything on Telegram's servers.

## Operator trust

A self-hosted relay concentrates trust in whoever runs it. On a given instance
the operator can, by design:

- Set `ANON3ANON_LOG_LEVEL=debug`, which logs message text, captions, usernames,
  and raw Telegram user IDs to stdout.
- Read the database and the `ANON3ANON_PSEUDONYM_KEY` together. With both:
  - `relays.origin_seal` decrypts to the sender's real chat ID, so any reply
    thread can be traced to an account.
  - The keyed reference is a deterministic keyed hash, not a per-row salted one,
    so the operator can confirm whether a *specific suspected* Telegram ID matches
    a stored `sender_ref`.

The pseudonym key protects against theft of the database *alone*. It does not
place the sender's identity beyond the reach of the person operating the service.
Use an instance run by someone you would trust with that.

## Data stored

SQLite database at `ANON3ANON_DATABASE_PATH` (default `/data/anon3anon.db`).
Timestamps are Unix seconds. "Keyed ref" = `HMAC-SHA256(chatID)` truncated to 16
bytes, hex-encoded; stable per key, not reversible. "Sealed" = `AES-256-GCM` with
a fresh random nonce per write; reversible **only** with the key.

### `users` - one row per registered recipient

| Column | Contents |
|--------|----------|
| `tg_user_id` | Recipient's Telegram user ID, **in the clear** |
| `chat_id` | Recipient's chat ID with the bot, **in the clear** |
| `link_token` | Random 64-bit token (base64url), the `?start=` value in the personal link |
| `created_at` | When `/start` first registered the account |

Only people who ran `/start` have a row here. Pure senders do not.

### `sessions` - where a sender's next first-contact message goes

| Column | Contents |
|--------|----------|
| `sender_ref` | Keyed ref of the sender's chat ID |
| `owner_user_id` | Recipient's Telegram user ID, in the clear |
| `updated_at` | Bumped on every inbound message from that sender |

### `relays` - reply routing for already-delivered messages

| Column | Contents |
|--------|----------|
| `dest_ref` | Keyed ref of the chat the copy was delivered to |
| `dest_msg_id` | Message ID of the delivered copy |
| `origin_ref` | Keyed ref of the chat a reply must go back to |
| `origin_seal` | Sealed (encrypted) origin chat ID - decryptable with the key, because a reply has to actually be delivered to that chat |
| `owner_user_id` | Recipient's Telegram user ID, in the clear |
| `created_at` | When the mapping was written |

Written for both directions of a conversation.

### `blocks` - per-recipient sender bans

| Column | Contents |
|--------|----------|
| `owner_user_id` | Recipient's Telegram user ID, in the clear |
| `sender_ref` | Keyed ref of the blocked sender's chat ID |
| `created_at` | When the block was created |

### `message_rates` - rate-limit counters

| Column | Contents |
|--------|----------|
| `sender_ref` | Keyed ref of the sender's chat ID |
| `recipient_id` | Recipient's Telegram user ID, in the clear |
| `bucket` | Time-window index |
| `count` | Messages in that window |

## Data NOT stored

- **Message content** - text, captions, photos, files, voice, video, stickers.
  None of it touches the database; it flows through `copyMessage`.
- **Anonymous senders' Telegram user IDs or chat IDs in the clear.** Only keyed
  refs, plus the reversible `origin_seal` in `relays` needed to deliver replies.
- **Usernames and display names** of anyone, senders or recipients.
- **Per-message timestamps or logs of individual messages.** Only the coarse
  `updated_at` / `created_at` on the rows above.
- **Any mapping from `link_token` to who has been given the link.** The bot does
  not know who holds a link until they use it.

### Logs

At `info` (default) a message produces one structured line with: update ID,
keyed `chat_ref`, chat type, message ID, a coarse kind (`text` / `command` /
`photo` / etc.), whether it is a reply, and whether it has media. **No text, no
username, no raw ID.** At `debug` an additional line carries the raw chat ID,
user ID, username, and message text - for diagnosis only.

The health endpoints (`/healthz`, `/readyz`) return only `ok` or `not ready` and
expose no data.

## Retention

A background sweep runs every `ANON3ANON_RETENTION_SWEEP_INTERVAL` (default `1h`)
and deletes rows older than `ANON3ANON_RETENTION_AGE` (default `720h`, i.e. 30
days). Set either to `0` to disable the sweep entirely.

| Table | Deleted when | Note |
|-------|--------------|------|
| `sessions` | `updated_at` older than the cutoff | Each inbound message bumps `updated_at`, so an actively used conversation is not pruned |
| `relays` | `created_at` older than the cutoff | **Not** bumped by activity. A reply mapping always expires `RETENTION_AGE` after it was created; after that, replying to that old delivered message no longer routes |
| `blocks` | `created_at` older than the cutoff | **A block is forgotten after `RETENTION_AGE`.** The banned sender can reach the recipient again unless re-blocked. Set `RETENTION_AGE=0` to keep blocks permanently |
| `message_rates` | window older than the cutoff | Only swept while rate limiting is enabled |

`users` rows are **never** pruned by retention. A registered recipient persists
until `/delete`.

## Deletion - `/delete`

Irreversible, no confirmation prompt, effective immediately. For a registered
recipient it runs one transaction that removes, matched by both the caller's
recipient ID and the keyed ref of their own chat (their footprint as a sender to
others):

- `sessions` where they are the owner **or** the sender
- `relays` where they are the owner **or** either endpoint
- `blocks` they created **or** blocks against them
- `message_rates` where they are the recipient **or** the sender
- their `users` row

So `/delete` erases both the account and the caller's traces as an anonymous
sender in other people's conversations.

`/delete` only acts for someone who has a `users` row (ran `/start`). A person
who has only ever *sent* anonymous messages has nothing for `/delete` to remove;
their `sessions` / `relays` / `message_rates` rows disappear instead via the
counterpart's `/delete`, the recipient's `/revoke`, the sender's own `/stop`, or
retention.

## Revocation - `/revoke`

For a registered recipient, `/revoke`:

1. **Rotates `link_token`.** The old personal link stops working; anyone who
   saved it can no longer start a conversation.
2. **Clears every `session` pointing at this recipient.** Senders who already had
   an open conversation must open a fresh link to write again.
3. **Clears every `relay` for this recipient, both directions.** Reply threads
   established before the revoke are cut - which also means the recipient loses
   the ability to answer messages received before the revoke. The two are the
   same rows; revocation is treated as the stronger promise.

All three steps are idempotent; a failed `/revoke` is completed by running it
again. `/revoke` does not touch `blocks` - existing bans survive a revoke (until
retention expires them).

## Related sender-side controls

- **`/stop`** clears the sender's own `session` and the `relays` for that one
  conversation only. Conversations that sender has with other recipients are
  untouched.
- **`/block`** (recipient replies with it to a delivered message) inserts a
  `blocks` row for that `(recipient, sender_ref)` pair. Further messages from that
  sender to that recipient are refused; other senders are unaffected. The block
  expires with retention unless `RETENTION_AGE=0`.

## See also

- [configuration.md](configuration.md) - `ANON3ANON_PSEUDONYM_KEY`,
  `ANON3ANON_LOG_LEVEL`, retention and rate-limit variables
- [deployment.md](deployment.md) - keeping the key off the data volume, backups
