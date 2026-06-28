# anon3anon [![Build Status](https://github.com/nightnoryu/anon3anon/actions/workflows/check-go.yml/badge.svg)](https://github.com/nightnoryu/anon3anon/actions/workflows/check-go.yml)

Telegram bot for anonymous messages.

## Local development

Prerequisites:

1. mise
2. docker
3. docker compose

Build the project with mise:

```shell
mise run
```

After that, copy the `compose.override.example.yml` to `compose.override.yml` and set the environment variables:

```yaml
services:
  anon3anon:
    environment:
      ANON3ANON_TELEGRAM_BOT_TOKEN: 123:ABC # The token for your bot, obtained from t.me/BotFather
      ANON3ANON_OWNER_CHAT_ID: 123 # ID of your chat with your bot
```

> [!NOTE]
> When launching the bot for the first time, leave `ANON3ANON_OWNER_CHAT_ID` empty and write a message to the bot. It will print the chat ID in the logs and after that you can set it up.

And you're set! Use `docker compose` to manage the application:

```shell
# Start
docker compose up -d

# Restart to apply changes
docker restart anon3anon

# Stop
docker compose down
```
