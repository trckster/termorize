<?php

namespace Termorize\Commands;

use Termorize\Commands\AbstractCommand;
use Longman\TelegramBot\Exception\TelegramException;
use Longman\TelegramBot\Request;
use Termorize\Services\Translator;

class TranslateCommand extends AbstractCommand
{
    public function process(): void
    {
        try {
            $translator = new Translator();
            $translationText = $translator->translate($this->update->getMessage()->getText());

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => $translationText
            ]);
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}
