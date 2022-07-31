<?php
require_once __DIR__ .'/../../vendor/autoload.php';

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
            }
        } catch (Longman\TelegramBot\Exception\TelegramException $e){
            echo $e->getMessage();
        }

    }
}