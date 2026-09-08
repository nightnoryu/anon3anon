# Changelog

## v1.0.0

- sender identifiers are pseudonymized at rest: `sessions`, `relays`, `blocks`, and `message_rates` now store a keyed
  reference derived from the sender's chat ID instead of the ID itself, so the database alone no longer links a
  conversation to a Telegram account
- `ANON3ANON_PSEUDONYM_KEY` is now required for pseudonymisation
- message text, usernames, and raw user IDs are no longer logged at the default level; they moved to `debug`
- `ANON3ANON_LOG_LEVEL` added (`debug`, `info`, `warn`, `error`; default `info`)

## v0.4.0

- `/delete` command added for recipients to erase their account and all associated data
- `contact`, `location`/`venue`, `story`, and shared-users/chat messages are now rejected instead of relayed, in both directions
- commands now match only at the start of a message (`/delete` no longer fires from a mid-text mention)
- a failed message delivery no longer consumes rate-limit quota
- the retention sweep now also prunes `blocks` and `message_rates`

## v0.3.0

- health and liveness probes added for container
- rate limiting now limits only the senders, not the owners
- `/help` command added to print commands info
- `/stop` command added for senders to exit current session
- bot restricted only to private chats
- `/revoke` now purges all active session created with old link

## v0.2.0

- rate limiting added per sender / recipient
- `/block` command added to block senders in scope of one recipient

## v0.1.0

Initial release
