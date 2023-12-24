<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Helpers\KeyboardHelper;
use Termorize\Models\VocabularyItem;
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
        $message = $this->update->getMessage()->getText();
        $translation = $this->translator->translate($message);

        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $translation->translation_text,
        ]);

        if (str_word_count($message) <= 5) {
            $this->vocabularyService->save($translation, $this->update->getMessage()->getFrom()->getId());
            $vocabularyItemId = VocabularyItem::query()
                ->where('translation_id', $translation->id)
                ->where('user_id', $this->update->getMessage()->getFrom()->getId())
                ->first()
                ->id;

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Перевод сохранён для дальнейшего изучения',
                'reply_markup' => KeyboardHelper::makeButton('Удалить из словарного запаса',
                    'deleteWord', [
                        'vocabularyItemId' => $vocabularyItemId,
                    ]),
            ]);
        } else {
            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Текст, введенный вами очень длинный, хотите ли вы сохранить его?',
                'reply_markup' => KeyboardHelper::makeButton('Сохранить в словарный запас',
                    'addWord', [
                        'translationId' => $translation->id,
                    ]),
            ]);
        }
    }
}
