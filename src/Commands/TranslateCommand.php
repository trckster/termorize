<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Exception\TelegramException;
use Longman\TelegramBot\Request;
use Termorize\Services\Translator;

class TranslateCommand
{
    public static function execute(string $text, string $chatId)
    {
        try {
            $translator = new Translator;
            $translationText = $translator->translate($text);
            Request::sendMessage([
                'chat_id' => $chatId,
                'text' => $translationText
            ]);
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
}