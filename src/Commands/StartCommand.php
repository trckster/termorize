<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class StartCommand extends AbstractCommand
{
    public function process(): void
    {
        $this->reply("Привет!\nОтправь мне любое слово и я его переведу. Или посмотри список команд: /help");
    }
}
