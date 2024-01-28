<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Termorize\Models\PendingTask;
use Termorize\Models\Translation;
use Termorize\Models\User;
use Termorize\Models\VocabularyItem;

class SendQuestion
{
    public function handle(PendingTask $pendingTask)
    {
        $params = json_decode($pendingTask->parameters, true);
        $userId = $params['chat_id'];
        $user = User::query()
            ->where('id', '=', $userId)
            ->first();

        $chatId = $user->chat()->id;

        $vocabularyItemId = VocabularyItem::query()
            ->where('id', '=', $params['user_id'])
            ->first()
            ->translation_id;

        $translationText = Translation::query()
            ->where('id', '=', $vocabularyItemId)
            ->first()
            ->translation_text;

        Request::sendMessage([
            'chat_id' => $chatId,
            'text' => $translationText,
        ]);
    }
}
