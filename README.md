# anon3anon

Telegram bot for anonymous messages.

## Self-hosting

There are 2 options for self-hosting the bot: via docker and k8s.

> [!NOTE]
> When launching the bot for the first time, leave `ANON3ANON_OWNER_CHAT_ID` empty and write a message to the bot. It
> will print the chat ID in the logs and after that you can set it up.

### Docker

Using `docker run`:

```shell
docker run -d \
  --name anon3anon \
  -e ANON3ANON_TELEGRAM_BOT_TOKEN=123:ABC \
  -e ANON3ANON_OWNER_CHAT_ID=123 \
  ghcr.io/nightnoryu/anon3anon:latest
```

Using `docker compose`, create a `compose.yml`:

```yaml
services:
  anon3anon:
    image: ghcr.io/nightnoryu/anon3anon
    environment:
      ANON3ANON_TELEGRAM_BOT_TOKEN: 123:ABC # The token for your bot, obtained from t.me/BotFather
      ANON3ANON_OWNER_CHAT_ID: 123 # ID of your chat with your bot
```

then run:

```shell
docker compose up -d
```

### K8s

Manifests live in `k8s/`. Copy `k8s/secret.example.yaml` to `k8s/secret.yaml` and fill in the bot token and owner chat
ID, then apply:

```shell
kubectl apply -k k8s/
```

This deploys a single-replica `Deployment` using the official image (`ghcr.io/nightnoryu/anon3anon:latest`).

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
> When launching the bot for the first time, leave `ANON3ANON_OWNER_CHAT_ID` empty and write a message to the bot. It
> will print the chat ID in the logs and after that you can set it up.

And you're set! Use `docker compose` to manage the application:

```shell
# Start
docker compose up -d

# Build & restart to apply changes
mise run
docker restart anon3anon

# Stop
docker compose down
```
