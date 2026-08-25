# anon3anon

Telegram bot for anonymous messages.

## Local Development

Prerequisites:

- [mise](https://mise.jdx.dev)
- Docker with docker-compose-plugin

### First launch

```shell
git clone https://github.com/nightnoryu/anon3anon
cd anon3anon

# Set the env
cp compose.override.example.yml compose.override.yml
$EDITOR compose.override.yml

mise run
docker compose up -d
```

## License

Distributed under the MIT License. See [License](/LICENSE) for more information.
