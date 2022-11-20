<?php

namespace Termorize\Services;

use Termorize\Commands\StartCommand;
use Termorize\Commands\TranslateCommand;
use Termorize\Commands\DefaultCommand;
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Exception\TelegramException;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try {
            $message = $update->getMessage();
            $chatId = $update->getMessage()->getChat()->getId();
            $text = $message->getText();

            switch($text) {
                case '/start':
                    $command = new StartCommand();
                    $command->process($chatId);
//                    StartCommand::execute($chatId);
//                    break;

                case $text[0] != '/':
                    $command = new TranslateCommand();
//                    TranslateCommand::execute($text, $chatId);
                    break;
                default:
                    $command = new DefaultCommand();
//                    DefaultCommand::execute($chatId);
            }

        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
