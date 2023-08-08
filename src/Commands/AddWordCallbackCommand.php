<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\Translation;
use Termorize\Models\VocabularyItem;
use Termorize\Services\VocabularyItemService;

class AddWordCallbackCommand extends AbstractCallbackCommand
{
    public function process(): void
    {
        $userId = $this->callbackQuery->getFrom()->getId();
        $callback_data = json_decode($this->callbackQuery->getData(), true);

        $service = new VocabularyItemService();
        $service->save(Translation::query()->find($callback_data['data']['translationId'])->first(), $userId);

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => 'Слово добавлено в словарный запас',
        ]);

    }
}