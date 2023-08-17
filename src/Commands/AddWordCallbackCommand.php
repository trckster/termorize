<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\Translation;
use Termorize\Services\VocabularyItemService;

class AddWordCallbackCommand extends AbstractCallbackCommand
{
    public function process(): void
    {
        $userId = $this->update->getCallbackQuery()->getFrom()->getId();

        $service = new VocabularyItemService();
        $translationToSave = Translation::query()->find($this->callbackData['data']['translationId'])->first();
        $service->save($translationToSave, $userId);

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => 'Слово добавлено в словарный запас',
        ]);

    }
}
