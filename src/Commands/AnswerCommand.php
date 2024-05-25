<?php

namespace Termorize\Commands;

use Termorize\Models\Question;

class AnswerCommand extends AbstractCommand
{
    private function giveVerdict(string $message): void
    {
        $this->reply($message, [
            'reply_to_message_id' => $this->update->getMessage()->getMessageId(),
            'parse_mode' => 'HTML',
        ]);
    }

    public function process(): void
    {
        /** @var Question $question */
        $question = Question::query()
            ->with('vocabularyItem.translation')
            ->where('chat_id', $this->update->getMessage()->getChat()->getId())
            ->where('message_id', $this->update->getMessage()->getReplyToMessage()->getMessageId())
            ->where('is_answered', false)
            ->first();

        if (!$question) {
            $this->giveVerdict('Вы уже ответили на этот вопрос!');

            return;
        }

        $vocabularyItem = $question->vocabularyItem;
        $expectedAnswer = $question->is_original
            ? $vocabularyItem->translation->translation_text
            : $vocabularyItem->translation->original_text;

        $answer = mb_strtolower($this->update->getMessage()->getText());
        $expectedAnswer = mb_strtolower($expectedAnswer);

        $verdict = "Неправильно, правильный ответ: <b>{$expectedAnswer}</b>\n";

        switch (mb_levenshtein($answer, $expectedAnswer)) {
            case 0:
                $vocabularyItem->update([
                    'knowledge' => min(100, $vocabularyItem->knowledge + 20),
                ]);
                $verdict = 'Правильный ответ! ';
                break;

            case 1:
                $vocabularyItem->update([
                    'knowledge' => min(100, $vocabularyItem->knowledge + 10),
                ]);
                $verdict = "Почти, правильный ответ: <b>{$vocabularyItem->translation->original_text}</b>\n";
                break;

            default:
                $vocabularyItem->update([
                    'knowledge' => max(0, $vocabularyItem->knowledge - 10),
                ]);
        }

        $emoji = match (true) {
            $vocabularyItem->knowledge >= 100 => '✅',
            $vocabularyItem->knowledge >= 80 => '🔝',
            $vocabularyItem->knowledge >= 60 => '🔥',
            $vocabularyItem->knowledge >= 40 => '📈',
            $vocabularyItem->knowledge >= 20 => '♻️',
            $vocabularyItem->knowledge > 0 => '↘️️',
            default => '❌',
        };

        $this->giveVerdict($verdict . "\n$emoji Текущее знание - <b>{$vocabularyItem->knowledge}%</b> $emoji");
        $question->update(['is_answered' => true]);
    }
}
