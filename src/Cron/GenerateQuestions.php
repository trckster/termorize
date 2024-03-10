<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Illuminate\Support\Facades\Log;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Models\UserSetting;
use Termorize\Tasks\SendQuestion;

class GenerateQuestions implements CronCommand
{
    public function handle(): void
    {
        $users = User::with('settings', 'vocabularyItems')->get();
        foreach ($users as $user) {
            if (!$user->is_bot) {
                $userSetting = $user->settings ??= UserSetting::createDefaultSetting($user);
                if ($userSetting->learns_vocabulary) {
                    $this->generateDayTasks($user);
                }
            }
        }
    }

    private function generateDayTasks(User $user): void
    {
        if ($user->vocabularyItems->where('knowledge', '<', 100)->random()) {
            $vocabularyItem = $user->vocabularyItems->where('knowledge', '<', 100)->random();
            PendingTask::query()->create([
                'status' => PendingTaskStatus::Pending,
                'method' => SendQuestion::class . '::execute',
                'parameters' => json_encode([
                    'user_id' => $user->id,
                    'vocabulary_item_id' => $vocabularyItem->id,
                ]),
                'scheduled_for' => Carbon::now(),
            ]);
        }
    }
}
