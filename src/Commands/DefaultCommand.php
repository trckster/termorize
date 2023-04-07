<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;

class DefaultCommand extends AbstractCommand
{
    public function process(): void
    {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Такой команды нет, попробуйте ввести другую',
            ]);
    }
}
