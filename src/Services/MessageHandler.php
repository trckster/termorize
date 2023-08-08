<?php

namespace Termorize\Services;

use Longman\TelegramBot\Entities\CallbackQuery;
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Exception\TelegramException;
use Termorize\Commands\DefaultCommand;
use Termorize\Commands\DeleteWordCallbackCommand;
use Termorize\Commands\StartCommand;
use Termorize\Commands\TranslateCommand;

class MessageHandler
{
    public function handle(Update $update): void
    {
        try {
           if($update->getMessage() !== null){
               $message = $update->getMessage();
               $text = $message->getText();

               if (empty($text)) {
                   $command = new StartCommand();
               } elseif ($text === '/start') {
                   $command = new StartCommand();
               } else {
                   if ($text[0] != '/') {
                       $command = new TranslateCommand();
                   } else {
                       $command = new DefaultCommand();
                   }
               }
               $command->setUpdate($update);
               $command->process();
           } else {
               if($update->getCallbackQuery() !== null)
               {
                   $this->handleCallback($update->getCallbackQuery());
               }
           }

        } catch (TelegramException $e) {
            echo $e->getMessage();
        }

    }
    private function handleCallback(CallbackQuery $callbackQuery){

        $callback_data = json_decode($callbackQuery->getData(), true);
        var_dump($callback_data);
            if($callback_data['callback'] === "deleteWord"){
                echo 'in';
                $callbackCommand = new DeleteWordCallbackCommand();
            }

            $callbackCommand->setCallbackQuery($callbackQuery);
            $callbackCommand->process();
        }
}
