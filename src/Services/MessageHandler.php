<?php

namespace Termorize\Services;

use Longman\TelegramBot\Entities\Update;
use Termorize\Commands\AddWordCallbackCommand;
use Termorize\Commands\AnswerCommand;
use Termorize\Commands\DefaultCommand;
use Termorize\Commands\DeleteWordCallbackCommand;
use Termorize\Commands\SetQuestionsCountCommand;
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
            $text = explode(' ', $text)[0];
            $command = match ($text) {
                '/start' => new StartCommand,
                '/toggle_questions' => new ToggleQuestionsSettingCommand,
                '/set_questions' => new SetQuestionsCountCommand,
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

        $callbackCommand = match ($callbackData['callback']) {
            'deleteWord' => new DeleteWordCallbackCommand,
            'addWord' => new AddWordCallbackCommand,
        };

        $callbackCommand->setUpdate($update);
        $callbackCommand->parseCallbackData();
        $callbackCommand->process();
    }
}
