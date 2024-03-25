<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Request;
use Termorize\Helpers\KeyboardHelper;
use Termorize\Models\VocabularyItem;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;
use Termorize\Models\User;

class UserAnswerCommand extends AbstractCommand
{
    public function process(): void
    {
        $user = User::query()->get()->where('username', $this->update->getMessage()->getChat()->getUsername());
    }
}
