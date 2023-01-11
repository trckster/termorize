<?php

namespace Termorize\Commands;

use Termorize\Commands\AbstractCommand;
use Longman\TelegramBot\Request;
use Longman\TelegramBot\Exception\TelegramException;

class DefaultCommand extends AbstractCommand
{
    public function process(): void
    {
        try {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Такой команды нет, попробуйте ввести другую'
            ]);
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
