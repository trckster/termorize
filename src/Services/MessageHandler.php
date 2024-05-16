<?php

namespace Termorize\Services;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Exception\TelegramException;
use Termorize\Commands\AddWordCallbackCommand;
use Termorize\Commands\AnswerCommand;
use Termorize\Commands\DefaultCommand;
use Termorize\Commands\DeleteWordCallbackCommand;
use Termorize\Commands\StartCommand;
use Termorize\Commands\TranslateCommand;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try {
            if ($update->getMessage() !== null) {
                $this->handleMessage($update);
            } else {
                if ($update->getCallbackQuery() !== null) {
                    $this->handleCallback($update);
                }
            }

        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }

    private function handleMessage(Update $update): void
    {
        $message = $update->getMessage();
        $text = $message->getText();

        if (empty($text)) {
            $command = new StartCommand();
        } elseif ($text === '/start') {
            $command = new StartCommand();
        } else {
            if ($text[0] != '/') {
                if ($message->getReplyToMessage() !== null) {
                    $command = new AnswerCommand();
                } else {
                    $command = new TranslateCommand();
                }
            } else {
                $command = new DefaultCommand();
            }
        }
        $command->setUpdate($update);
        $command->process();
    }

    private function handleCallback(Update $update): void
    {
        $callbackData = json_decode($update->getCallbackQuery()->getData(), true);
        if ($callbackData['callback'] === 'deleteWord') {
            $callbackCommand = new DeleteWordCallbackCommand();
        } elseif ($callbackData['callback'] === 'addWord') {
            $callbackCommand = new AddWordCallbackCommand();
        }

        $callbackCommand->setUpdate($update);
        $callbackCommand->parseCallbackData();
        $callbackCommand->process();
    }
}
