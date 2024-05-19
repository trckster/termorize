<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Models\VocabularyItem;
use Termorize\Services\Logger;
use Termorize\Tasks\SendQuestion;

class GenerateQuestions implements CronCommand
{
    public function handle(): void
    {
        Logger::info('Generating daily questions...');
        $questionsCount = 0;
        $usersCount = 0;

        $users = User::with('settings', 'vocabularyItems')->get();
        /** @var User $user */
        foreach ($users as $user) {
            if (!$user->is_bot) {
                $userSetting = $user->getOrCreateSettings();
                if ($userSetting->learns_vocabulary) {
                    $newQuestions = $this->generateDayTasks($user);

                    if ($newQuestions > 0) {
                        $usersCount++;
                        $questionsCount += $newQuestions;
                    }
                }
            }
        }

        Logger::info("Generated $questionsCount questions for $usersCount users.");
    }

    private function generateDayTasks(User $user): int
    {
        $learnToday = $user->vocabularyItems
            ->where('knowledge', '<', 100)
            ->shuffle()
            ->slice(0, $user->settings->questions_count);

        $tasksScheduled = $learnToday->count();

        $learnToday->each(function (VocabularyItem $item) use ($user) {
            $this->scheduleQuestion($user->id, $item->id);
        });

        return $tasksScheduled;
    }

    private function scheduleQuestion(int $userId, int $vocabularyItemId): void
    {
        $randomTime = env('DEBUG', false)
            ? Carbon::now()
            : Carbon::today()->addMinutes(rand(0, 60 * 24));

        PendingTask::query()->create([
            'status' => PendingTaskStatus::Pending,
            'method' => SendQuestion::class . '::execute',
            'parameters' => [
                'user_id' => $userId,
                'vocabulary_item_id' => $vocabularyItemId,
            ],
            'scheduled_for' => $randomTime,
        ]);
    }
}
