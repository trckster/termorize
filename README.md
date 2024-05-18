# About

This is a telegram bot, written in PHP. It's goal to teach and
develop programming skills working on project, that is close to real.

# Run locally

Set environment variables inside .env and run docker compose:

```shell
docker compose up --build
```

# Deploy

Build & push images and dockerhub will automatically deploy them afterward.

```bash
docker build -t trckster/termorize-bot .
docker build -t trckster/termorize-cron . -f cron/Dockerfile
docker push trckster/termorize-bot
docker push trckster/termorize-cron
```

# Todo list

1. Ability to choose questions per day.
2. Multi-language support.
3. Translations mass upload.
4. Fix .env creation inside Dockerfiles