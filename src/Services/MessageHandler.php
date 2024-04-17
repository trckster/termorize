<?php

namespace Termorize\Services;

use GuzzleHttp\Psr7\Message;
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Exception\TelegramException;
use Longman\TelegramBot\Request;
use Termorize\Commands\AddWordCallbackCommand;
use Termorize\Commands\DefaultCommand;
use Termorize\Commands\DeleteWordCallbackCommand;
use Termorize\Commands\StartCommand;
use Termorize\Commands\TranslateCommand;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Enums\UserStatus;
use Termorize\Models\PendingTask;
use Termorize\Models\Translation;
use Termorize\Models\UserChat;
use Termorize\Models\UserSetting;
use Termorize\Models\VocabularyItem;

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

        $userChat = UserChat::query()->get()->where('chat_id', $update->getMessage()->getChat()->getId())->first;

        $userSetting = UserSetting::query()
            ->get()
            ->where('user_id', $userChat->user_id)
            ->first;

        if (empty($text)) {
            $command = new StartCommand();
        } elseif ($text === '/start') {
            $command = new StartCommand();
        } else {
            if ($text[0] != '/') {
                    $command = new TranslateCommand();
                }
            else {
                $command = new DefaultCommand();
            }
        }
        $command->setUpdate($update);
        $command->process();
    }

    private function handleCallback(Update $update): void
    {
        $callback_data = json_decode($update->getCallbackQuery()->getData(), true);
        if ($callback_data['callback'] === 'deleteWord') {
            $callbackCommand = new DeleteWordCallbackCommand();
        } elseif ($callback_data['callback'] === 'addWord') {
            $callbackCommand = new AddWordCallbackCommand();
        }

        $callbackCommand->setUpdate($update);
        $callbackCommand->parseCallbackData();
        $callbackCommand->process();
    }

    private function handleReply(Update $update): void
    {
        $originMessageId = $update->getMessage()->getReplyToMessage()->getMessageId();

        $originMessage = \Termorize\Models\Message::query()
            ->where('id', $originMessageId)
            ->get();


        $translation = Translation::query()->where('original_text', $originMessage->text)->first();
        $vocabularyItem = VocabularyItem::query()->where('translation_id', $translation->id)->first();
        $pendingTask = PendingTask::query()->where('vocabulary_item_id', $vocabularyItem->id)->first();

        $message = $update->getMessage();
        $text = $message->getText();
        if ($text === $translation->original_text) {
            $vocabularyItem->update(['knowledge' => $translation->knowledge + 20]);

            Request::sendMessage([
                'chat_id' => $update->getMessage()->getChat()->getId(),
                'text' => "Правильный ответ! Текущее знание - `$vocabularyItem->knowledge%`"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Success
            ]);
        }

        if (levenshtein($text, $translation->original_text) === 1 ) {
            $vocabularyItem->update(['knowledge' => $translation->knowledge + 10]);

            Request::sendMessage([
                'chat_id' => $update->getMessage()->getChat()->getId(),
                'text' => "Почти, правильный ответ:$translation->translation_text\n Текущее знание - `$vocabularyItem->knowledge%`"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Success
            ]);
        }

        if (levenshtein($text, $translation->original_text) > 1 ) {
            if ($translation->knowledge != 0) {
                $vocabularyItem->update(['knowledge' => $translation->knowledge - 10]);
            }

            Request::sendMessage([
                'chat_id' => $update->getMessage()->getChat()->getId(),
                'text' => "Неправильно, правильный ответ:$translation->translation_text\n Текущее знание - `$vocabularyItem->knowledge%`"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Failed
            ]);
        }

    }
}
