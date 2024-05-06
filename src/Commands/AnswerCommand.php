<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;

class AnswerCommand extends AbstractCommand
{
    public function process(): void
    {
        $originMessageText = $this->update->getMessage()->getReplyToMessage()->getText();

        $translation = Translation::query()->where('translation_text', $originMessageText)->first();

        $vocabularyItem = VocabularyItem::query()
            ->where('translation_id', $translation->id)
            ->where('user_id', $this->update->getMessage()->getFrom()->getId())
            ->first();

        $message = $this->update->getMessage();
        $text = $message->getText();
        if ($text === $translation->original_text) {
            $toAdd = 20;

            if ($vocabularyItem->knowledge + 20 > 100) {
                $toAdd = 100 - $vocabularyItem->knowledge;
            }

            $vocabularyItem->update(['knowledge' => $vocabularyItem->knowledge + $toAdd]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'reply_to_message_id' => $this->update->getMessage()->getMessageId(),
                'parse_mode' => 'HTML',
                'text' => "Правильный ответ! Текущее знание - <b>{$vocabularyItem->knowledge}%</b>",
            ]);
        } elseif (levenshtein($text, $translation->original_text) === 1) {
            $toAdd = 20;

            if ($vocabularyItem->knowledge + 10 > 100) {
                $toAdd = 100 - $vocabularyItem->knowledge;
            }

            $vocabularyItem->update(['knowledge' => $vocabularyItem->knowledge + $toAdd]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'reply_to_message_id' => $this->update->getMessage()->getMessageId(),
                'parse_mode' => 'HTML',
                'text' => "Почти, правильный ответ:<b>{$translation->original_text}</b>\n Текущее знание - <b>{$vocabularyItem->knowledge}%</b>",
            ]);

        } elseif (levenshtein($text, $translation->original_text) > 1) {
            if ($translation->knowledge > 0) {
                $newValue = max(0, $translation->knowledge - 10);

                $vocabularyItem->update(['knowledge' => $newValue - 10]);
            }

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'reply_to_message_id' => $this->update->getMessage()->getMessageId(),
                'parse_mode' => 'HTML',
                'text' => "Неправильно, правильный ответ:<b>{$translation->original_text}</b>\n Текущее знание - <b>{$vocabularyItem->knowledge}%</b>",
            ]);

        }
    }
}
