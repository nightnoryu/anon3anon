# anon3anon

Telegram bot for anonymous messages 🎭✨

---

[![Go Report Card](https://goreportcard.com/badge/github.com/nightnoryu/anon3anon)](https://goreportcard.com/report/github.com/nightnoryu/anon3anon)

## Building for local development

Prerequisites:

1. Linux
2. Git
3. Docker

Firstly, clone the repository into your `$GOPATH`:

```shell
mkdir -p $GOPATH/src/github.com/nightnoryu
cd $GOPATH/src/github.com/nightnoryu

git clone git@github.com:nightnoryu/anon3anon.git
cd anon3anon
```

Then build the binary:

```shell
bin/a3abrewkit build
```

This script will download a [brewkit build system](https://github.com/ispringtech/brewkit) binary and put it in the `bin` directory of the project.

After that, copy the `docker-compose.override.example.yml` to `docker-compose.override.yml` and set the environment variables:

```yaml
services:
  anon3anon:
    environment:
      TELEGRAM_BOT_TOKEN: 123:ABC # The token for your bot, obtained from t.me/BotFather
      OWNER_CHAT_ID: 123 # ID of your chat with your bot
```

And you're set! Use the provided `docker compose` wrapper script to manage the application:

```shell
# Start
bin/a3acompose up -d

# Restart to apply changes
docker restart anon3anon

# Stop
bin/a3acompose down
```
