<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Models\UserSetting;
use Termorize\Services\Logger;
use Termorize\Tasks\SendQuestion;

class GenerateQuestions implements CronCommand
{
    public function handle(): void
    {
        Logger::info('Generating daily questions...');
        $questionsCount = 0;

        $users = User::with('settings', 'vocabularyItems')->get();
        foreach ($users as $user) {
            if (!$user->is_bot) {
                $userSetting = $user->settings ??= UserSetting::createDefaultSetting($user);
                if ($userSetting->learns_vocabulary) {
                    $questionsCount += $this->generateDayTasks($user) ? 1 : 0;
                }
            }
        }

        Logger::info("Questions generated: $questionsCount");
    }

    private function generateDayTasks(User $user): bool
    {
        $vocabularyToLearn = $user->vocabularyItems->where('knowledge', '<', 100);

        if ($vocabularyToLearn->isEmpty()) {
            return false;
        }

        $vocabularyItem = $vocabularyToLearn->random();
        PendingTask::query()->create([
            'status' => PendingTaskStatus::Pending,
            'method' => SendQuestion::class . '::execute',
            'parameters' => [
                'user_id' => $user->id,
                'vocabulary_item_id' => $vocabularyItem->id,
            ],
            'scheduled_for' => Carbon::today()->addHours(rand(10, 22)),
        ]);

        return true;
    }
}
