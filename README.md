# About

This is a telegram bot, written in PHP. It's goal to teach and
develop programming skills working on project, that is close to real.

# Set up

### CRON

"0 0 * * * php /src/Cron/GenerateQuestions.php"

Enter this in crontab in order to generate pending tasks for bot users.

"* * * * * php /src/Cron/CloseQuestions.php"

Enter this in crontab in order to send pending tasks for bot users.

# Build

```bash
docker build -t trckster/termorize .
docker push trckster/termorize
```

# Run locally

Set environment variables inside .env and run docker compose:

```shell
docker compose up --build
```
