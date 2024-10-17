<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Termorize\Enums\Language;
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

        $username = $user->username ?? $userId;
        Logger::info("Sending new question to the $username");

        /** @var UserChat $userChat */
        $userChat = UserChat::query()->where('user_id', $user->id)->first();

        /** @var VocabularyItem $vocabularyItem */
        $vocabularyItem = VocabularyItem::query()->with('translation')->find($vocabularyItemId);
        $translation = $vocabularyItem->translation;

        $sendOriginalWord = (bool) rand(0, 1);

        $wordToSend = $sendOriginalWord ? $translation->original_text : $translation->translation_text;
        $expectedAnswer = $sendOriginalWord ? $translation->translation_text : $translation->original_text;
        $expectedAnswerLanguage = $sendOriginalWord ? $translation->translation_lang : $translation->original_lang;

        $message = $vocabularyItem->knowledge >= 100 ? 'Повторение!' : 'Ежедневное упражнение:';

        $answerLength = mb_strlen($expectedAnswer);
        $lettersClarification = self::clarificationNeeded($vocabularyItemId, $wordToSend) ? "\n(в ответе содержится $answerLength символов)" : '';

        $languageClarification = '';
        if ($expectedAnswerLanguage !== Language::ru) {
            $translateTo = $expectedAnswerLanguage->getName();
            $languageClarification = ' на ' . mb_strtolower($translateTo);
        }

        $response = Request::sendMessage([
            'chat_id' => $userChat->chat_id,
            'parse_mode' => 'HTML',
            'text' => $message . "\n\nПеревидите$languageClarification слово <b>$wordToSend</b>$lettersClarification\n\n(ответ отправьте реплаем на это сообщение)",
        ]);

        if (!$response->isOk()) {
            Logger::info("Request failed for user $username: " . $response->getDescription());
            $user->getOrCreateSettings()->update(['learns_vocabulary' => false]);

            return;
        }

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
