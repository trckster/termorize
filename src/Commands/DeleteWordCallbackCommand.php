<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\VocabularyItem;

class DeleteWordCallbackCommand extends AbstractCallbackCommand
{
    public function process(): void
    {
        $userId = $this->update->getCallbackQuery()->getFrom()->getId();

        /** @var VocabularyItem|null $item */
        $item = VocabularyItem::query()
            ->with('questions')
            ->find($this->callbackData['data']['vocabularyItemId']);

        if (!$item) {
            Request::sendMessage([
                'chat_id' => $userId,
                'text' => 'Слово уже удалено',
            ]);

            return;
        }

        $item->questions()->delete();
        $item->delete();

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => 'Слово удалено из словарного запаса',
        ]);
    }
}
