CREATE TABLE IF NOT EXISTS users (
    tg_user_id INTEGER PRIMARY KEY,
    chat_id    INTEGER NOT NULL,
    link_token TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    sender_chat_id INTEGER PRIMARY KEY,
    owner_user_id  INTEGER NOT NULL REFERENCES users (tg_user_id),
    updated_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS relays (
    dest_chat_id   INTEGER NOT NULL,
    dest_msg_id    INTEGER NOT NULL,
    origin_chat_id INTEGER NOT NULL,
    owner_user_id  INTEGER NOT NULL,
    created_at     INTEGER NOT NULL,
    PRIMARY KEY (dest_chat_id, dest_msg_id)
);

CREATE TABLE IF NOT EXISTS message_rates (
    sender_id    INTEGER NOT NULL,
    recipient_id INTEGER NOT NULL,
    bucket       INTEGER NOT NULL,
    count        INTEGER NOT NULL,
    PRIMARY KEY (sender_id, recipient_id, bucket)
);
