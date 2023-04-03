<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;

class TranslateCommand extends AbstractCommand
{
    public function process(): void
    {
        $translator = new Translator();
        $translation = $translator->translate($this->update->getMessage()->getText());
        Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $translation->translation_text,
        ]);

        $vocabularyItem = new VocabularyItemService();
        $vocabularyItem->save($translation, $this->update->getMessage()->getFrom()->getId());
    }
}
