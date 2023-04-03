FROM php:8.2-cli

RUN apt update && apt upgrade -y

RUN apt-get install -y libzip-dev zip

RUN docker-php-ext-install zip pdo pdo_mysql
RUN docker-php-ext-enable pdo_mysql


COPY --from=composer /usr/bin/composer /usr/bin/composer

COPY composer.json ./

WORKDIR /var/www

COPY composer.* ./

RUN composer install --no-ansi --no-dev --no-interaction --no-plugins --no-progress --no-scripts --optimize-autoloader; \
    composer clearcache

ADD src ./src
ADD index.php .env ./

CMD php index.php

