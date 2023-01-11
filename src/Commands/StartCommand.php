<?php

namespace Termorize\Commands;

use Termorize\Commands\AbstractCommand;
use Longman\TelegramBot\Request;
use Longman\TelegramBot\Exception\TelegramException;

class StartCommand extends AbstractCommand
{
    public function process(): void
    {
        try {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Отправь мне любое слово и я его переведу.'
            ]);
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
