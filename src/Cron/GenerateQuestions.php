<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Models\UserSetting;
use Termorize\Tasks\SendQuestion;

class GenerateQuestions implements CronCommand
{
    public function handle(): void
    {
        $users = User::with('settings', 'vocabularyItems')->get();
        foreach ($users as $user) {
            $userSetting = $user->settings;
            if ($userSetting->learns_vocabulary) {
                $this->generateDayTasks($user);
            }
        }
    }

    private function generateDayTasks(User $user): void
    {
        $vocabularyItem = $user->vocabularyItems->where('knowledge', '<', 100)->random();
        PendingTask::query()->create([
            'status' => PendingTaskStatus::Pending,
            'method' => SendQuestion::class . '::execute',
            'parameters' => [$user->id, $vocabularyItem->id],
            'scheduled_for' => Carbon::today()->addHours(rand(10, 22)),
        ]);
    }
}