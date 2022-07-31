<?php
require_once __DIR__ .'/../../vendor/autoload.php';
require_once __DIR__.'/Translator.php';

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;

class MessageHandler
{
    public function handle(\Longman\TelegramBot\Entities\Update $update){
        try {
            $message = $update->getMessage();
            $chat_id = $update->getMessage()->getChat()->getId();
            $text = $message->getText();

            if ($text == '/start')
            {
                Request::sendMessage([
                    'chat_id' => $chat_id,
                    'text' => "Отправь мне любое слово и я его переведу."
                ]);
            } else {
                $translation_text = Translator::translate($text);
                Request::sendMessage([
                    'chat_id' => $chat_id,
                    'text' => $translation_text
                ]);

            }
        } catch (Longman\TelegramBot\Exception\TelegramException $e){
            echo $e->getMessage();
        }

    }
}