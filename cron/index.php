<?php

require_once __DIR__ . '/../vendor/autoload.php';

use Longman\TelegramBot\Telegram;
use Termorize\Services\Kernel;

if ($argc < 2) {
    throw new Error('Not enough arguments');
}

$className = $argv[1];

$kernel = new Kernel;
$kernel->connectDatabase();

$classesInCron = scandir(getBasePath('src/Cron'));

new Telegram(env('BOT_API_KEY'), env('BOT_USERNAME'));

foreach ($classesInCron as $fileName) {
    if ($fileName === "$className.php") {
        $className = "Termorize\Cron\\$className";
        $command = new $className;
        $command->handle();
    }
}
