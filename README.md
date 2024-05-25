# About

Here are sources of Telegram bot, that can:

1. Translate words and save them
2. Every day send these words to you to help you memorize them

Bot: [@termorize_bot](https://t.me/termorize_bot)

# Run locally

Set environment variables inside .env and run docker compose:

```shell
docker compose up --build
```

# Deploy

Build & push images and dockerhub will automatically deploy them afterward.

```bash
./deploy.sh
```

# Todo list

1. Multi-language support.
2. Translations mass upload.
3. Fix .env creation inside Dockerfiles.
