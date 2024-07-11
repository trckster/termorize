# About

Here are sources of Telegram bot, that can:

1. Translate words and save them
2. Add custom words translations
3. Every day send these words to you to help you memorize them

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

1. Translations mass import.
2. Translations mass export.
3. Generate statistics every week & /stat
4. Fix .env creation inside Dockerfiles.
