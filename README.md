# anon3anon

[![Go Report Card](https://goreportcard.com/badge/github.com/nightnoryu/anon3anon)](https://goreportcard.com/report/github.com/nightnoryu/anon3anon)
[![GitHub License](https://img.shields.io/github/license/nightnoryu/anon3anon)](https://opensource.org/license/MIT)

Telegram bot for anonymous messages 🎭✨

Currently deployed and working at https://t.me/meme_me_a_meme_bot for my channel.

## Building for local development

Prerequisites:

1. Linux
2. Git
3. Docker
4. (optional) [BrewKit](https://github.com/ispringtech/brewkit)

Firstly, clone the repository into your `$GOPATH`:

```shell
mkdir -p $GOPATH/src/github.com/nightnoryu
cd $GOPATH/src/github.com/nightnoryu
git clone git@github.com:nightnoryu/anon3anon.git
cd anon3anon
```

Then build the project:

```shell
brewkit build

# Alternatively, if you don't want to use BrewKit, you can do it the old-fashioned way:
# go build -o ./bin/anon3anon ./cmd/anon3anon
```

After that, copy the `docker-compose.override.example.yml` to `docker-compose.override.yml` and set the environment variables:

```yaml
services:
  anon3anon:
    environment:
      ANON3ANON_TELEGRAM_BOT_TOKEN: 123:ABC # The token for your bot, obtained from t.me/BotFather
      ANON3ANON_OWNER_CHAT_ID: 123 # ID of your chat with your bot
```

And you're set! Use `docker compose` to manage the application:

```shell
# Start
docker compose up -d

# Restart to apply changes
docker restart anon3anon

# Stop
docker compose down
```
