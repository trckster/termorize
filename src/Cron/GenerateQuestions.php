<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Services\Logger;
use Termorize\Tasks\SendQuestion;

class GenerateQuestions implements CronCommand
{
    public function handle(): void
    {
        Logger::info('Generating daily questions...');
        $questionsCount = 0;

        $users = User::with('settings', 'vocabularyItems')->get();
        /** @var User $user */
        foreach ($users as $user) {
            if (!$user->is_bot) {
                $userSetting = $user->getOrCreateSettings();
                if ($userSetting->learns_vocabulary) {
                    $questionsCount += $this->generateDayTasks($user);
                }
            }
        }

        Logger::info("Questions generated: $questionsCount");
    }

    private function generateDayTasks(User $user): int
    {
        $vocabularyToLearn = $user->vocabularyItems->where('knowledge', '<', 100);

        if ($vocabularyToLearn->isEmpty()) {
            return 0;
        }

        $vocabularyItem = $vocabularyToLearn->random();
        PendingTask::query()->create([
            'status' => PendingTaskStatus::Pending,
            'method' => SendQuestion::class . '::execute',
            'parameters' => [
                'user_id' => $user->id,
                'vocabulary_item_id' => $vocabularyItem->id,
            ],
            'scheduled_for' => env('DEBUG', false) ? Carbon::now() : Carbon::today()->addHours(rand(10, 22)),
        ]);

        return 1;
    }
}
