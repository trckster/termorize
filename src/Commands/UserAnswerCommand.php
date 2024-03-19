<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Helpers\KeyboardHelper;
use Termorize\Models\VocabularyItem;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;

class UserAnswerCommand extends AbstractCommand
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
        #TODO Add user answer handling
    }
}
