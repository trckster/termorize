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

CI is not set up. Clone latest version manually and run it with docker compose.

# Todo list

1. Translations mass import.
2. Translations mass export.
3. Generate improved statistics every week.
4. Fix .env creation inside Dockerfiles.
