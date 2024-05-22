# About

This is a telegram bot, written in PHP. It's goal to teach and
develop programming skills working on project, that is close to real.

# Run locally

Set environment variables inside .env and run docker compose:

```shell
docker compose up --build
```

# Deploy

Build & push images and dockerhub will automatically deploy them afterwards.

```bash
./deploy.sh
```

# Todo list

1. Multi-language support.
2. Fix problem with questions for words that are translated equally
3. Translations mass upload.
4. Fix .env creation inside Dockerfiles
