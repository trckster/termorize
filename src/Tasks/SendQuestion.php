<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Longman\TelegramBot\Telegram;
use Termorize\Models\Translation;
use Termorize\Models\TranslationTasks;
use Termorize\Models\User;
use Termorize\Models\UserChat;
use Termorize\Models\VocabularyItem;
use Termorize\Services\Logger;

class SendQuestion
{
    public static function execute(array $questionData): void
    {
        $userId = $questionData['user_id'];
        $vocabularyItemId = $questionData['vocabulary_item_id'];

        /** @var User $user */
        $user = User::query()->find($userId);
        Logger::info("Sending new question to the {$user->username}");

        /** @var UserChat $userChat */
        $userChat = UserChat::query()->where('user_id', $user->id)->first();

        /** @var VocabularyItem $vocabularyItem */
        $vocabularyItem = VocabularyItem::query()->find($vocabularyItemId);
        // Might be united to the one query
        /** @var Translation $translation */
        $translation = Translation::query()->find($vocabularyItem->translation_id);

        $botUsername = env('BOT_USERNAME');
        $botApiKey = env('BOT_API_KEY');

        new Telegram($botApiKey, $botUsername);

        Request::sendMessage([
            'chat_id' => $userChat->chat_id,
            'text' => $translation->translation_text,
        ]);

        TranslationTasks::query()->create([
            'user_id' => $userId,
            'vocabulary_item_id' => $vocabularyItemId,
        ]);
    }
}
