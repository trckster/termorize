FROM golang:1.27.3-alpine3.24@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build

WORKDIR /src

ENV GOMAXPROCS=2
ENV GOFLAGS=-p=2

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/termorize ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/generate-exercises ./cmd/cron/generate.exercises.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/delete-expired-guests ./cmd/cron/delete-expired-guests/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/classify-words ./cmd/cron/classify-words

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

ARG SUPERCRONIC_VERSION=v0.2.51
ARG SUPERCRONIC_SHA256=cc232108374e8e2b537045f2d5eaa1229f619db314c3ba6a14d38b35dc0aa123

WORKDIR /app

RUN apk add --no-cache ca-certificates=20261020-r0 tzdata=2026e-r0 curl=8.23.0-r0 ffmpeg=8.1.3-r0 && \
    curl -fsSL -o /usr/local/bin/supercronic "https://github.com/aptible/supercronic/releases/download/${SUPERCRONIC_VERSION}/supercronic-linux-amd64" && \
    echo "${SUPERCRONIC_SHA256}  /usr/local/bin/supercronic" | sha256sum -c - && \
    chmod +x /usr/local/bin/supercronic

COPY --from=build /out/termorize ./termorize
COPY --from=build /out/generate-exercises ./generate-exercises
COPY --from=build /out/delete-expired-guests ./delete-expired-guests
COPY --from=build /out/classify-words ./classify-words
COPY src/data/migrations ./src/data/migrations
COPY cmd/cron/crontab ./crontab

EXPOSE 8080

CMD ["./termorize"]
