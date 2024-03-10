FROM php:8.2-cli

WORKDIR /app

RUN apt update && apt upgrade -y

RUN apt install -y libzip-dev zip

RUN docker-php-ext-install zip pdo pdo_mysql
RUN docker-php-ext-enable pdo_mysql

COPY --from=composer /usr/bin/composer /usr/bin/composer
COPY composer.* ./
RUN composer install --no-ansi --no-dev --no-interaction --no-plugins --no-progress --no-scripts --optimize-autoloader; \
    composer clearcache

ADD . .
RUN touch .env # Fix later

CMD php index.php

