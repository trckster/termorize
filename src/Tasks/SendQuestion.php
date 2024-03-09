<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Longman\TelegramBot\Telegram;
use Termorize\Models\PendingTask;
use Termorize\Models\Translation;
use Termorize\Models\User;
use Termorize\Models\UserChat;
use Termorize\Models\VocabularyItem;

class SendQuestion
{
    public static function execute(PendingTask $pendingTask) : void
    {
        $params = json_decode($pendingTask->parameters, true);

        $userId = $params['user_id'];
        $vocabularyItemId = $params['vocabulary_item_id'];

        $user = User::query()
            ->where('id', '=', $userId)
            ->first();

        $chatId = UserChat::query()->where('user_id', '=', $user->id)->first()->chat_id;

        $vocabularyItem = VocabularyItem::query()
            ->find($vocabularyItemId)
            ->where('id', '=', $vocabularyItemId)
            ->first();

        $translationText = Translation::query()
            ->where('id', '=', $vocabularyItem->translation_id)
            ->first()
            ->translation_text;

        $botUsername = env('BOT_USERNAME');
        $botApiKey = env('BOT_API_KEY');

        new Telegram($botApiKey, $botUsername);

        Request::sendMessage([
            'chat_id' => $chatId,
            'text' => $translationText,
        ]);
    }
}
