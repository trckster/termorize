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
            ->where(PendingTask::$scheduled_for, '=', Carbon::now())
            ->get()
            ->toArray();

        foreach($pendingTasks as $pendingTask)
        {
            $method = $pendingTask->method;

            call_user_func($method);

        }
    }
}