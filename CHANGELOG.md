# Changelog

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
