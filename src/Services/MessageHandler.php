<?php

namespace Termorize\Services;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Longman\TelegramBot\Exception\TelegramException;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try {
            $message = $update->getMessage();
            $chatId = $update->getMessage()->getChat()->getId();
            $text = $message->getText();

            switch($text){
                case "/start":
                    Request::sendMessage([
                        'chat_id' => $chatId,
                        'text' => "Отправь мне любое слово и я его переведу."
                    ]);
                    break;

                default:
                    $translator = new Translator;
                    $translationText = $translator->translate($text);
                    Request::sendMessage([
                        'chat_id' => $chatId,
                        'text' => $translationText
                    ]);
                    break;
            }
        } catch (TelegramException $e){
            echo $e->getMessage();
        }

    }
}