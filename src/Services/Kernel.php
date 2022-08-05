<?php

namespace Termorize\Services;

use Longman\TelegramBot\Telegram;
use Longman\TelegramBot\Exception\TelegramException;

class Kernel
{

    public function run()
    {
        $bot_username = $_ENV['BOT_USERNAME'];
        $bot_api_key = $_ENV['BOT_API_KEY'];


        try {
            $telegram = new Telegram($bot_api_key, $bot_username);


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