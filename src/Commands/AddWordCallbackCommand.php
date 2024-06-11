<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Models\Translation;
use Termorize\Services\VocabularyItemService;

class AddWordCallbackCommand extends AbstractCallbackCommand
{
    private readonly VocabularyItemService $vocabularyService;

    public function __construct()
    {
        $this->vocabularyService = new VocabularyItemService;
    }

    public function process(): void
    {
        $userId = $this->update->getCallbackQuery()->getFrom()->getId();

        /** @var Translation $translation */
        $translation = Translation::query()->find($this->callbackData['data']['translationId']);

        $message = 'Слово добавлено в словарный запас';

        $item = $this->vocabularyService->save($translation, $userId);
        if (!$item) {
            $message = 'Слово уже есть в словарном запасе';
        }

        Request::sendMessage([
            'chat_id' => $userId,
            'text' => $message,
        ]);
    }
}
