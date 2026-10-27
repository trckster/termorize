FROM node:24.22.1-alpine3.24@sha256:06d7adf3e0ef61a13f1d28682726366f97a15aa1888c934d3ecec86c33918b53 AS build

WORKDIR /app

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN npm install --global pnpm@12.15.0

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .

ARG VITE_API_URL=/api
ARG VITE_BOT_USERNAME
ARG VITE_SENTRY_DSN

ENV VITE_API_URL=$VITE_API_URL
ENV VITE_BOT_USERNAME=$VITE_BOT_USERNAME
ENV VITE_SENTRY_DSN=$VITE_SENTRY_DSN

RUN test -n "$VITE_BOT_USERNAME"
RUN pnpm build

FROM nginx:1.30.6-alpine3.24@sha256:47928bf67eb62a1863cb08d9ecb52d3e704304ba614eaee17884204677702951

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
