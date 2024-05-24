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

1. With some probability send one question to repeat words with 100%. 
2. Multi-language support.
3. Fix problem with questions for words that are translated equally.
4. Translations mass upload.
5. Fix .env creation inside Dockerfiles.
