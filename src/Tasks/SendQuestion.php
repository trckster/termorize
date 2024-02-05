<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Termorize\Models\PendingTask;
use Termorize\Models\Translation;
use Termorize\Models\User;

class SendQuestion
{
    public function handle(PendingTask $pendingTask)
    {
        $params = json_decode($pendingTask->parameters, true);

        $userId = $params['user_id'];
        $vocabularyItemId = $params['vocabulary_item_id'];

        $user = User::query()
            ->where('id', '=', $userId)
            ->first();

        $chatId = $user->chat()->first()->id;

        $vocabularyItem = $user->vocabularyItems()
            ->where('id', '=', $vocabularyItemId)
            ->first();

        $translationText = Translation::query()
            ->where('id', '=', $vocabularyItem->translation_id)
            ->first()
            ->translation_text;

        Request::sendMessage([
            'chat_id' => $chatId,
            'text' => $translationText,
        ]);
    }
}
