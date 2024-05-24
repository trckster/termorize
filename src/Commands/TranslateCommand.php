<?php

namespace Termorize\Commands;

use Termorize\Helpers\KeyboardHelper;
use Termorize\Models\Translation;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;

class TranslateCommand extends AbstractCommand
{
    private Translator $translator;
    private VocabularyItemService $vocabularyService;

    public function __construct()
    {
        $this->translator = new Translator;
        $this->vocabularyService = new VocabularyItemService;
    }

    public function process(): void
    {
        $message = $this->update->getMessage()->getText();

        $translation = $this->translator->translate($message);

        $this->reply($translation->translation_text);

        if (str_word_count($message) <= 5) {
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
