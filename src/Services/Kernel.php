<?php

namespace Termorize\Services;

use Longman\TelegramBot\Telegram;
use Longman\TelegramBot\Exception\TelegramException;

class Kernel
{
    public function run()
    {
        $botUsername = $_ENV['BOT_USERNAME'];
        $botApiKey = $_ENV['BOT_API_KEY'];

        try {
            $telegram = new Telegram($botApiKey, $botUsername);

            $telegram->useGetUpdatesWithoutDatabase();

            $response = $telegram->handleGetUpdates();
            $result = $response->getResult();
            $handler = new MessageHandler;

            foreach ($result as $update) {
                $handler->handle($update);
            }
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}