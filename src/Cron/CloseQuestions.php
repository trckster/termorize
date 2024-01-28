<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;

class CloseQuestions implements CronCommand
{
    public function handle()
    {
        $pendingTasks = PendingTask::query()
            ->where('status', '=', 'Pending')
            ->toArray();

        foreach($pendingTasks as $pendingTask) {
            $pendingTaskTime = $pendingTask->scheduled_for;
            if ($pendingTaskTime->diffInMinutes(Carbon::now()) <= 10) {
                $method = $pendingTask->method;
                call_user_func($method);
            }
        }
    }
}
