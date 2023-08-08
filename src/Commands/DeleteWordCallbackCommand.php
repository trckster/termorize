<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\VocabularyItem;

class DeleteWordCallbackCommand extends AbstractCallbackCommand
{
    public function process(): void
    {
        $userId = $this->callbackQuery->getFrom()->getId();
        $callback_data = json_decode($this->callbackQuery->getData(), true);

        VocabularyItem::query()->find($callback_data['data']['vocabularyItemId'])->delete();

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => 'Слово удалено из словарного запаса',
        ]);

    }
}
