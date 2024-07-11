<?php

namespace Termorize\Commands;

use Termorize\Helpers\KeyboardHelper;
use Termorize\Models\Translation;
use Termorize\Services\TranslationService;
use Termorize\Services\VocabularyItemService;

class TranslateCommand extends AbstractCommand
{
    private TranslationService $translationService;
    private VocabularyItemService $vocabularyService;

    public function __construct()
    {
        $this->translationService = new TranslationService;
        $this->vocabularyService = new VocabularyItemService;
    }

    public function process(): void
    {
        $message = $this->update->getMessage()->getText();

        $userSettings = $this->loadUser()->getOrCreateSettings();
        $translation = $this->translationService->translate($message, $userSettings->language);

        $resultingTranslation = $translation->original_text === mb_strtolower($message)
            ? $translation->translation_text
            : $translation->original_text;

        $this->reply($resultingTranslation);

        if (count(explode(' ', $resultingTranslation)) <= 5) {
            $this->addVocabularyItem($translation);
            return;
        }

        $this->reply('Текст достаточно длинный, хотите ли вы сохранить его в словарный запас для последующего повторения?', [
            'reply_markup' => KeyboardHelper::makeButton('Сохранить в словарный запас',
                'addWord', [
                    'translationId' => $translation->id,
                ]),
        ]);
    }

    private function addVocabularyItem(Translation $translation): void
    {
        $vocabularyItem = $this->vocabularyService->save($translation, $this->update->getMessage()->getFrom()->getId());

        if (!$vocabularyItem) {
            $this->reply('Слово уже есть в вашем словарном запасе');
            return;
        }

        $this->reply('Перевод сохранён для дальнейшего изучения', [
            'reply_markup' => KeyboardHelper::makeButton('Удалить из словарного запаса',
                'deleteWord', [
                    'vocabularyItemId' => $vocabularyItem->id,
                ]),
        ]);
    }
}
