<?php

namespace Termorize\Commands;

use Termorize\Services\VocabularyItemService;
use Throwable;

class DeleteVocabularyItemCommand extends AbstractCommand
{
    private readonly VocabularyItemService $vocabularyService;

    public function __construct()
    {
        $this->vocabularyService = new VocabularyItemService;
    }

    public function process(): void
    {
        $word = $this->getClearedMessage();

        try {
            $this->vocabularyService->deleteItem($this->update->getMessage()->getFrom()->getId(), $word);
            $this->reply("Слово удалено из словарного запаса");
        } catch (Throwable $e) {
            $this->reply("Слово не найдено");
        }
    }
}