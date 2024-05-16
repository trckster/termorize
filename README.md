# About

This is a telegram bot, written in PHP. It's goal to teach and
develop programming skills working on project, that is close to real.

# Run locally

Set environment variables inside .env and run docker compose:

```shell
docker compose up --build
```

# Deploy

```bash
docker build -t trckster/termorize-bot .
docker build -t trckster/termorize-cron . -f cron/Dockerfile
docker push trckster/termorize-bot
docker push trckster/termorize-cron
```
