<?php

namespace Termorize\Services;

use Termorize\Commands;
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Exception\TelegramException;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try
        {
            $message = $update->getMessage();
            $chatId = $update->getMessage()->getChat()->getId();
            $text = $message->getText();

            switch($text){
                case '/start':
                    Commands\StartCommand::execute($chatId);
                    break;

                case $text[0] != '/':
                    Commands\TranslateCommand::execute($text, $chatId);
                    break;
                default:
                    Commands\DefaultCommand::execute($chatId);
            }
        } catch (TelegramException $e){
            echo $e->getMessage();
        }

    }
}