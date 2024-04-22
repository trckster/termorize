<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Models\PendingTask;
use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;

class AnswerCommand extends AbstractCommand
{
    public function process(): void
    {
        $originMessageText = $this->update->getMessage()->getReplyToMessage()->getText();

        $translation = Translation::query()->where('translation_text', $originMessageText)->first();

        $vocabularyItem = VocabularyItem::query()
            ->get()
            ->where('translation_id', $translation->id)
            ->where('user_id', $this->update->getMessage()->getFrom()->getId())
            ->first();

        var_dump($translation);

        $pendingTask = PendingTask::query()
            ->get()
            ->where('vocabulary_item_id', $vocabularyItem->id)
            ->first();

        $message = $this->update->getMessage();
        $text = $message->getText();
        if ($text === $translation->original_text) {
            $toAdd = 20;

            if ($vocabularyItem->knowledge + 20 > 100)
            {
                $toAdd = 100 - $vocabularyItem->knowledge;
            }

            $vocabularyItem->update(['knowledge' => $vocabularyItem->knowledge + $toAdd]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => "Правильный ответ! Текущее знание - $vocabularyItem->knowledge%"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Success
            ]);
        }

        if (levenshtein($text, $translation->original_text) === 1 ) {
            $vocabularyItem->update(['knowledge' => $vocabularyItem->knowledge + 10]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => "Почти, правильный ответ:$translation->translation_text\n Текущее знание - $vocabularyItem->knowledge%"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Success
            ]);
        }

        if (levenshtein($text, $translation->original_text) > 1 ) {
            if ($translation->knowledge != 0) {
                $vocabularyItem->update(['knowledge' => $vocabularyItem->knowledge - 10]);
            }

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => "Неправильно, правильный ответ:$translation->original_text\n Текущее знание - $vocabularyItem->knowledge%"
            ]);

            $pendingTask->update([
                'status' => PendingTaskStatus::Failed
            ]);
        }
    }
}
