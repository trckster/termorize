#!/bin/bash
docker build -t trckster/termorize-bot .
docker build -t trckster/termorize-cron . -f cron/Dockerfile
docker push trckster/termorize-bot
docker push trckster/termorize-cron