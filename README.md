# :performing_arts: anon3anon [![Github release](https://img.shields.io/github/release/nightnoryu/anon3anon.svg)](https://github.com/nightnoryu/anon3anon/releases) [![Build Status](https://github.com/nightnoryu/anon3anon/actions/workflows/check-go.yml/badge.svg)](https://github.com/nightnoryu/anon3anon/actions/workflows/check-go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/nightnoryu/anon3anon)](https://goreportcard.com/report/github.com/nightnoryu/anon3anon)

Telegram bot for anonymous messages.

Currently running at https://t.me/meme_me_a_meme_bot for my channel.

## Local development

Prerequisites:

1. Git
2. Docker
3. [bkit](https://github.com/nightnoryu/bkit)

Firstly, clone the repository into your `$GOPATH`:

```shell
mkdir -p $GOPATH/src/github.com/nightnoryu
cd $GOPATH/src/github.com/nightnoryu

git clone git@github.com:nightnoryu/anon3anon.git
cd anon3anon
```

Then build the project:

```shell
bkit build
```

After that, copy the `docker-compose.override.example.yml` to `docker-compose.override.yml` and set the environment variables:

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
