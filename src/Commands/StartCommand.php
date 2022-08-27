<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Longman\TelegramBot\Exception\TelegramException;

class StartCommand
{
public static function execute(string $chatId) : void
{
    try{
        Request::sendMessage([
            'chat_id' => $chatId,
            'text' => "Отправь мне любое слово и я его переведу."
        ]);
    } catch (TelegramException $e){
        echo $e->getMessage();
    }
}
}