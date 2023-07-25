<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\VocabularyItem;
use Termorize\Services\VocabularyItemService;

class DeleteWordCallbackCommand extends AbstractCommand
{
    public function process(): void
    {
        $userId = $this->update->getCallbackQuery()->getFrom()->getId();

        $service = new VocabularyItemService();
       // $service->deleteLatestUserTranslation();

        Request::sendMessage([
            'chat_id' => $this->update->getCallbackQuery()->getMessage()->getChat()->getId(),
            'text' => 'Слово удалено из словарного запаса',
        ]);

    }
}