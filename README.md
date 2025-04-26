# anon3anon

Telegram bot for anonymous messages 🎭✨

## Building for local development

Prerequisites:

1. git
2. docker

Firstly, clone the repository:

```shell
git clone git@github.com:nightnoryu/anon3anon.git
```

Then build the binary:

```shell
./bin/a3abrewkit build
```

This script will download a [brewkit](https://github.com/ispringtech/brewkit) binary and put it in the `bin` directory of the project.

After that, copy the `docker-compose.override.example.yml` to `docker-compose.override.yml` and set the environment variables:

```yaml
services:
  anon3anon:
    environment:
      TELEGRAM_BOT_TOKEN: 123:ABC # The token for your bot, obtained from t.me/BotFather
      OWNER_CHAT_ID: 123 # ID of your chat with your bot
```

And you're set! Lastly, run the app:

```shell
./bin/a3acompose up -d
```

While making changes to the app, don't forget to restart the container:

```shell
docker restart anon3anon
```

When you're finished working on the project, bring it down using this command:

```shell
./bin/a3acompose down
```
