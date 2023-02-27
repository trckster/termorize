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
            $text = $message->getText();

            /*switch($text) {
                case '/start':
                    $command = new StartCommand();
                    break;

                case $text[0] != '/': // TODO: Fix bug here
                    $command = new TranslateCommand();
                    break;

                default:
                    $command = new DefaultCommand();
            }*/
            if($text === '/start')
            {
                $command = new StartCommand();
            } else {
                if($text[0] != '/')
                {
                    $command = new TranslateCommand();
                } else {
                    $command = new DefaultCommand();
                }
            }

            $command->setUpdate($update);
            $command->process();
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
