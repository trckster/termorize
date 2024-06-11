<?php

namespace Termorize\Commands;

use Termorize\Services\TranslationService;

class AddCustomVocabularyCommand extends AbstractCommand
{
    private readonly TranslationService $translationService;

    public function __construct()
    {
        $this->translationService = new TranslationService;
    }

    public function process(): void
    {
        $message = $this->getClearedMessage();

        $parts = explode(':', $message);

        if (count($parts) !== 2) {
            $this->reply('В сообщении должно быть ровно одно двоеточие.');
            return;
        }

        $translation = $this->translationService->saveCustomTranslation($parts[0], $parts[1]);
        $translation->vocabularyItems()->create([
            'user_id' => $this->update->getMessage()->getFrom()->getId(),
            'knowledge' => 0,
        ]);

        $this->reply("Ваш собственный перевод успешно сохранён!");
    }
}