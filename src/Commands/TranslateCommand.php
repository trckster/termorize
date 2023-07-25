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
        $message = $this->update->getMessage()->getText();
        $translation = $this->translator->translate($message);

        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $translation->translation_text,
        ]);

        if (str_word_count($message) <= 5) {
            $this->vocabularyService->save($translation, $this->update->getMessage()->getFrom()->getId());
            $keyboard = json_encode([
                "inline_keyboard" => [
                    [
                        ["text" => "Удалить из словарного запаса", "callback_data" => "deleteWord"]
                    ]
                ]
            ]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Перевод сохранён для дальнейшего изучения',
                'reply_markup' => $keyboard
            ]);
        } else {
            $keyboard = json_encode([
                "inline_keyboard" => [
                    [
                        ["text" => "Сохранить для дальнейшего обучения", "callback_data" => "addWord"]
                    ]
                ]
            ]);

            Request::sendMessage([
                'chat_id' => $this->update->getMessage()->getChat()->getId(),
                'text' => 'Текст, введенный вами очень длинный, хотите ли вы сохранить его?',
                'reply_markup' => $keyboard
            ]);
        }
    }
}
