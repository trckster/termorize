<?php

namespace Termorize\Services;

use Longman\TelegramBot\Entities\Update;
use Termorize\Commands\AddWordCallbackCommand;
use Termorize\Commands\AnswerCommand;
use Termorize\Commands\DefaultCommand;
use Termorize\Commands\DeleteWordCallbackCommand;
use Termorize\Commands\StartCommand;
use Termorize\Commands\ToggleQuestionsSettingCommand;
use Termorize\Commands\TranslateCommand;
use Throwable;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try {
            if ($update->getMessage() !== null) {
                $this->handleMessage($update);
            } elseif ($update->getCallbackQuery() !== null) {
                $this->handleCallback($update);
            }
        } catch (Throwable $e) {
            Logger::info($e->getMessage());
        }
    }

    private function handleMessage(Update $update): void
    {
        $message = $update->getMessage();
        $text = $message->getText();

        if (str_starts_with($text, '/')) {
            $command = match ($text) {
                '/start' => new StartCommand,
                '/toggle_questions' => new ToggleQuestionsSettingCommand,
                default => new DefaultCommand,
            };
        } elseif (empty($text)) {
            $command = new StartCommand;
        } elseif ($message->getReplyToMessage()) {
            $command = new AnswerCommand;
        } else {
            $command = new TranslateCommand;
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
