<?php

namespace Termorize\Commands;

class StartCommand extends AbstractCommand
{
    public function process(): void
    {
        $this->reply("Привет!\nОтправь мне любое слово и я его переведу. Или посмотри список команд: /help");
    }
}
