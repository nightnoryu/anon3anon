<!-- markdownlint-disable -->
<p align="center">
	<img src="https://github.com/user-attachments/assets/21484ffa-225b-4a99-88c3-1b6f95497b8a" width="180" title="anon3anon Logo">
</p>

<h1 align="center">anon3anon</h1>
<p align="center"><i>Talk without trading identities</i></p>

<p align="center">
    <a href="https://github.com/nightnoryu/anon3anon/releases"><img src="https://img.shields.io/github/release/nightnoryu/anon3anon.svg?cache-control=no-cache"></a>
    <a href="https://github.com/nightnoryu/anon3anon/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/anon3anon?cache-control=no-cache"></a>
    <a href="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/anon3anon/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>
</p>
<!-- markdownlint-enable -->

An anonymous contact relay for Telegram. anon3anon lets two people
communicate without exposing their Telegram identities to each other.
Share a personal link to receive anonymous messages; reply to continue the
conversation without either side seeing the other's identity.

## ✨ Features

- Personal links and two-way anonymous replies
- Media-friendly relays without Telegram forward attribution
- Link revocation, sender blocking, conversation exit, and account deletion
- Per-pair rate limits, optional recipient allow list, and pseudonymized sender
  records

## 💬 How it works

1. A recipient sends `/start` and shares the personal link
2. A sender opens it and messages normally
3. Replies to delivered messages keep the conversation anonymous in both directions

## 🚀 Run your own

The quickest path - a pre-built image and a volume:

```shell
docker run -d --name anon3anon \
  -e ANON3ANON_TELEGRAM_BOT_TOKEN=123:ABC \
  -e ANON3ANON_PSEUDONYM_KEY="$(openssl rand -base64 32)" \
  -v anon3anon-data:/data \
  ghcr.io/nightnoryu/anon3anon:latest
```

See [deployment documentation](docs/deployment.md) for Compose, Kubernetes,
and operational guidance.

## 📚 Documentation

- [Architecture](docs/architecture.md), [configuration](docs/configuration.md),
  [development](docs/development.md), and [privacy model](docs/privacy.md)
- [Changelog](CHANGELOG.md)

## 📜 License

Distributed under the MIT License. See [License](/LICENSE) for more information.
