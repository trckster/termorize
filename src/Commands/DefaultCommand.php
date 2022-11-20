<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Longman\TelegramBot\Exception\TelegramException;

class DefaultCommand
{
    public static function execute(string $chatId): void
    {
        try {
            Request::sendMessage([
                'chat_id' => $chatId,
                'text' => 'Такой команды нет, попробуйте ввести другую'
            ]);
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
