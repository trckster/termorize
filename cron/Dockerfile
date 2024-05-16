FROM php:8.3-cli

WORKDIR /termorize

RUN apt-get update && apt-get install cron -y

RUN apt install -y libzip-dev zip

RUN docker-php-ext-install zip pdo pdo_mysql
RUN docker-php-ext-enable pdo_mysql

COPY --from=composer /usr/bin/composer /usr/bin/composer
COPY composer.* ./
RUN composer install --no-ansi --no-dev --no-interaction --no-plugins --no-progress --no-scripts --optimize-autoloader; \
    composer clearcache

ADD . .
RUN touch .env # Fix later

RUN mv cron/crontab.termorize /etc/cron.d/termorize-cron
RUN chmod 0644 /etc/cron.d/termorize-cron

CMD printenv >> /etc/environment && cron -f