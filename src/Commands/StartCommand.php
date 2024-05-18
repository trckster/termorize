<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class StartCommand extends AbstractCommand
{
    public function process(): void
    {
        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => "Привет!\nОтправь мне любое слово и я его переведу. Или посмотри список команд: /help",
        ]);
    }
}
