<?php
require_once __DIR__ . '/vendor/autoload.php';
require_once __DIR__.'/src/Services/MessageHandler.php';
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;

$dotenv = Dotenv\Dotenv::createImmutable(__DIR__);
$dotenv->load();

$bot_username = $_ENV['BOT_USERNAME'];
$bot_api_key = $_ENV['BOT_API_KEY'];



try {
    $telegram = new Longman\TelegramBot\Telegram($bot_api_key, $bot_username);


    $telegram->useGetUpdatesWithoutDatabase();

    $response = $telegram->handleGetUpdates();
    $result = $response->getResult();
    $handler = new MessageHandler();


    foreach ($result as $update) {
        $handler->handle($update);
    }
} catch (Longman\TelegramBot\Exception\TelegramException $e) {
    echo $e->getMessage();
}
