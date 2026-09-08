CREATE TABLE IF NOT EXISTS users (
    tg_user_id INTEGER PRIMARY KEY,
    chat_id    INTEGER NOT NULL,
    link_token TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    sender_ref    TEXT PRIMARY KEY,
    owner_user_id INTEGER NOT NULL REFERENCES users (tg_user_id),
    updated_at    INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions (updated_at);

CREATE TABLE IF NOT EXISTS relays (
    dest_ref      TEXT    NOT NULL,
    dest_msg_id   INTEGER NOT NULL,
    origin_ref    TEXT    NOT NULL,
    origin_seal   BLOB    NOT NULL,
    owner_user_id INTEGER NOT NULL,
    created_at    INTEGER NOT NULL,
    PRIMARY KEY (dest_ref, dest_msg_id)
);

CREATE INDEX IF NOT EXISTS idx_relays_created_at ON relays (created_at);

CREATE TABLE IF NOT EXISTS blocks (
    owner_user_id INTEGER NOT NULL,
    sender_ref    TEXT    NOT NULL,
    created_at    INTEGER NOT NULL,
    PRIMARY KEY (owner_user_id, sender_ref)
);

CREATE INDEX IF NOT EXISTS idx_blocks_created_at ON blocks (created_at);

CREATE TABLE IF NOT EXISTS message_rates (
    sender_ref   TEXT    NOT NULL,
    recipient_id INTEGER NOT NULL,
    bucket       INTEGER NOT NULL,
    count        INTEGER NOT NULL,
    PRIMARY KEY (sender_ref, recipient_id, bucket)
);

CREATE INDEX IF NOT EXISTS idx_message_rates_bucket ON message_rates (bucket);
