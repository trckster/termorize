<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Exception\TelegramException;
use Longman\TelegramBot\Request;

class StartCommand extends AbstractCommand
{
    public function process(): void
    {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Отправь мне любое слово и я его переведу.',
            ]);
    }
}

