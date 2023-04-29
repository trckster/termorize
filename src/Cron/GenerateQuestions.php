<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;
use Termorize\Models\User;
use Termorize\Enums\PendingTaskStatus;

class GenerateQuestions implements CronCommand
{
    public function handle()
    {
        $users = User::all();
        foreach($users as $user)
        {
            $userSetting = $user->setting();
            if($userSetting->learns_vocabulary)
            {
                $vocabularyItems = $user->vocabularyItems()::all();
                foreach ($vocabularyItems as $item)
                {
                    if($item->knowledge < 100)
                    {
                        PendingTask::query()->create(
                            [
                                'status' => PendingTaskStatus::Pending,
                                'method' => Termorize\Tasks\SendQuestion::execute,
                                'parameters' => [$user->id, $item->id],
                                'scheduled_for' => Carbon::today()->addHours(rand(10, 22))
                            ]
                        );
                    }
                }
            }
        }
    }
}