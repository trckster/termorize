<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Termorize\Models\Question;
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
        $vocabularyItem = VocabularyItem::query()->with('translation')->find($vocabularyItemId);

        $response = Request::sendMessage([
            'chat_id' => $userChat->chat_id,
            'parse_mode' => 'HTML',
            'text' => "Ежедневное упражнение\n\nПеревидите слово <b>{$vocabularyItem->translation->translation_text}</b>\n\n(ответ отправьте реплаем на это сообщение)",
        ]);

        Question::query()->create([
            'chat_id' => $userChat->chat_id,
            'message_id' => $response->getResult()->getMessageId(),
            'vocabulary_item_id' => $vocabularyItemId,
            // TODO send both originals and translations
            'is_original' => false,
        ]);
    }
}
