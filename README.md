# anon3anon

*Talk without trading identities.*

<a href="https://github.com/nightnoryu/anon3anon/releases"><img src="https://img.shields.io/github/release/nightnoryu/anon3anon.svg?cache-control=no-cache"></a>
<a href="https://github.com/nightnoryu/anon3anon/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/anon3anon?cache-control=no-cache"></a>
<a href="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>

**anon3anon** is an anonymous contact relay for Telegram. It lets two people
communicate without exposing their Telegram identities to each other.

Share a personal link, get anonymous messages, reply to them in a threaded
conversation - in both directions, with no identity ever crossing between the two
sides.

## ✅ Features

- **Per-user personal links** - `t.me/<bot>?start=<token>`, with unguessable random tokens
- **Two-way threaded conversations** - the recipient replies to a delivered message and the answer goes back to the
  original anonymous sender, still anonymous in both directions
- **`/revoke`** rotates your link, kills the old one, and drops both the routing sessions opened through it and the
  reply threads already established, so senders who already had the link can no longer reach you
- **`/block`** as a reply to a delivered message stops that one anonymous sender from ever reaching you again
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
4. `/stop` leaves the conversation - neither new messages nor replies to already delivered ones go anywhere
   until you open a link again. Conversations with other recipients are unaffected

## 🚀 Run your own

The quickest path - a pre-built image and a volume:

```shell
docker run -d --name anon3anon \
  -e ANON3ANON_TELEGRAM_BOT_TOKEN=123:ABC \
  -e ANON3ANON_PSEUDONYM_KEY="$(openssl rand -base64 32)" \
  -v anon3anon-data:/data \
  ghcr.io/nightnoryu/anon3anon:latest
```

Compose and Kubernetes setups, backups, egress proxying, and the full
walkthrough are in **[docs/deployment.md](docs/deployment.md)**.

## 📚 Documentation

- **[Architecture](docs/architecture.md)** - domain model, sessions and relays, pseudonymization, why HMAC, why single-connection SQLite
- **[Configuration](docs/configuration.md)** - every `ANON3ANON_` environment variable, with defaults and notes
- **[Deployment](docs/deployment.md)** - self-hosting with Docker, Compose, or Kubernetes
- **[Development](docs/development.md)** - local setup with mise and Docker, the edit/rebuild loop, task list
- **[Privacy model](docs/privacy.md)** - what it protects and what it doesn't, what is stored, retention, deletion, revocation
- **[Changelog](CHANGELOG.md)** - release history

## 📜 License

Distributed under the MIT License. See [License](/LICENSE) for more information.
