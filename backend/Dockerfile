FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/termorize ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/generate-exercises ./cmd/cron/generate.exercises.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/delete-expired-guests ./cmd/cron/delete-expired-guests/main.go

FROM alpine:3.23

ARG SUPERCRONIC_VERSION=v0.2.44

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl ffmpeg && \
    curl -fsSL -o /usr/local/bin/supercronic "https://github.com/aptible/supercronic/releases/download/${SUPERCRONIC_VERSION}/supercronic-linux-amd64" && \
    chmod +x /usr/local/bin/supercronic

COPY --from=build /out/termorize ./termorize
COPY --from=build /out/generate-exercises ./generate-exercises
COPY --from=build /out/delete-expired-guests ./delete-expired-guests
COPY src/data/migrations ./src/data/migrations
COPY cmd/cron/crontab ./crontab

EXPOSE 8080

CMD ["./termorize"]
