<?php

namespace Termorize\Services;

use Longman\TelegramBot\Telegram;
use Longman\TelegramBot\Exception\TelegramException;

class Kernel
{
    public function run()
    {
        $botUsername = env('BOT_USERNAME');
        $botApiKey = env('BOT_API_KEY');

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