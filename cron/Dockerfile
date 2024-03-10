FROM php:8.2-cli

WORKDIR /termorize

RUN apt-get update && apt-get install cron -y

COPY src termorize/

COPY --from=composer /usr/bin/composer /usr/bin/composer

COPY ../composer.* ./

RUN composer install --no-ansi --no-dev --no-interaction --no-plugins --no-progress --no-scripts --optimize-autoloader; \
    composer clearcache

RUN touch .env # Fix later

ADD cron/crontab.termorize /etc/cron.d/termorize-cron

RUN chmod 0644 /etc/cron.d/termorize-cron

RUN apt install -y libzip-dev zip

RUN docker-php-ext-install zip pdo pdo_mysql
RUN docker-php-ext-enable pdo_mysql

CMD cron -f