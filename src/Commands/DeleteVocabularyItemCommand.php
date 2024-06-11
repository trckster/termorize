<?php

namespace Termorize\Commands;

use Illuminate\Support\Str;
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
        $message = $this->update->getMessage()->getText();

        $word = trim(Str::replaceStart('/delete_vocabulary', '', $message));

        try {
            $this->vocabularyService->deleteItem($this->update->getMessage()->getFrom()->getId(), $word);
            $this->reply("Слово удалено из словарного запаса");
        } catch (Throwable $e) {
            $this->reply("Слово не найдено");
        }
    }
}