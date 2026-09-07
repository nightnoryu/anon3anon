<p align="center">
  <a href="https://github.com/nightnoryu/anon3anon/releases"><img src="https://img.shields.io/github/release/nightnoryu/anon3anon.svg?cache-control=no-cache"></a>
  <a href="https://github.com/nightnoryu/anon3anon/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/anon3anon?cache-control=no-cache"></a>
  <a href="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>
</p>

Multi-tenant Telegram bot for anonymous messages. Available at [@anon3anon_bot](https://t.me/anon3anon_bot).

## ✅ Features

- Per-user personal links (`t.me/<bot>?start=<token>`), unguessable random tokens
- Two-way, threaded anonymous conversations (reply to a message to answer)
- `/mylink` to show your link again, `/revoke` to rotate it and kill the old one

## 🚀 Hosting

TODO

## 🛠 Local Development

### Prerequisites

- [mise](https://mise.jdx.dev)
- Docker with docker-compose-plugin

### First launch

```shell
git clone https://github.com/nightnoryu/anon3anon
cd anon3anon

# Configure the environment
cp compose.override.example.yml compose.override.yml
$EDITOR compose.override.yml

mise run      # Build the binary
mise run dev  # Spins up docker container
```

## 📜 License

Distributed under the MIT License. See [License](/LICENSE) for more information.
