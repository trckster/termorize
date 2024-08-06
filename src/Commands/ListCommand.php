<?php

namespace Termorize\Commands;

use Generator;
use Illuminate\Database\Eloquent\Collection;
use Termorize\Enums\Language;
use Termorize\Models\VocabularyItem;

class ListCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = $this->loadUser();

        $items = $user->vocabularyItems()
            ->with('translation')
            ->orderByDesc('knowledge')
            ->get();

        $message = '';
        $messagesCount = 0;

        foreach ($this->getMessageParts($items) as $part) {
            $newPossibleMessage = $message . $part;

            if (mb_strlen($newPossibleMessage) > self::MAX_MESSAGE_LENGTH) {
                $this->reply($message);
                $messagesCount++;
                $message = $part;
            } else {
                $message = $newPossibleMessage;
            }

            if ($messagesCount >= self::MAX_MESSAGES_AT_ONCE) {
                $this->reply('У вас слишком большой словарный запас, не влезает в 10 сообщений. Попробуйте /export');
            }
        }

        $this->reply($message);
    }

    private function getMessageParts(Collection $items): Generator
    {
        if ($items->isEmpty()) {
            yield 'Your vocabulary is empty!';
        }

        /** @var VocabularyItem $item */
        foreach ($items as $item) {
            $ruText = $item->translation->original_lang === Language::ru
                ? $item->translation->original_text
                : $item->translation->translation_text;

            $foreignText = $item->translation->original_lang === Language::ru
                ? $item->translation->translation_text
                : $item->translation->original_text;

            yield "$ruText — $foreignText — {$item->knowledge}%\n";
        }
    }
}
