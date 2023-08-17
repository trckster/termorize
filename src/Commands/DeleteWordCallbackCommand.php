<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\VocabularyItem;

class DeleteWordCallbackCommand extends AbstractCallbackCommand
{
    public function process(): void
    {
        $userId = $this->update->getCallbackQuery()->getFrom()->getId();

        VocabularyItem::query()->find($this->callbackData['data']['vocabularyItemId'])->delete();

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => 'Слово удалено из словарного запаса',
        ]);

    }
}
