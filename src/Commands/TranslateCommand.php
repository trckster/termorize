<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;

class TranslateCommand extends AbstractCommand
{
    private Translator $translator;
    private VocabularyItemService $vocabularyService;

    public function __construct()
    {
        $this->translator = new Translator();
        $this->vocabularyService = new VocabularyItemService();
    }

    public function process(): void
    {
        $translation = $this->translator->translate($this->update->getMessage()->getText());

        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $translation->translation_text,
        ]);

        $this->vocabularyService->save($translation, $this->update->getMessage()->getFrom()->getId());
    }
}
