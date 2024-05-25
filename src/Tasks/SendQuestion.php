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
        $translation = $vocabularyItem->translation;

        $sendOriginalWord = (bool)rand(0, 1);

        $wordToSend = $sendOriginalWord ? $translation->original_text : $translation->translation_text;

        $message = $vocabularyItem->knowledge >= 100 ? "Повторение!" : "Ежедневное упражнение:";

        $answerLength = mb_strlen($sendOriginalWord ? $translation->translation_text : $translation->original_text);
        $clarification = self::clarificationNeeded($vocabularyItemId, $wordToSend) ? "\n(в ответе содержится $answerLength символов)" : "";

        $response = Request::sendMessage([
            'chat_id' => $userChat->chat_id,
            'parse_mode' => 'HTML',
            'text' => $message . "\n\nПеревидите слово <b>{$wordToSend}</b>$clarification\n\n(ответ отправьте реплаем на это сообщение)",
        ]);

        Question::query()->create([
            'chat_id' => $userChat->chat_id,
            'message_id' => $response->getResult()->getMessageId(),
            'vocabulary_item_id' => $vocabularyItemId,
            'is_original' => $sendOriginalWord,
        ]);
    }

    private static function clarificationNeeded(int $vocabularyItemId, string $word): bool
    {
        return VocabularyItem::query()
            ->where('id', '!=', $vocabularyItemId)
            ->where(function ($query) use ($word) {
                $query->whereRelation('translation', 'original_text', $word)
                    ->orWhereRelation('translation', 'translation_text', $word);
            })
            ->exists();
    }
}
